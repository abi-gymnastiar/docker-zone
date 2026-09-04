package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Container   string   `yaml:"container" json:"container"`
	Description string   `yaml:"description" json:"description"`
	Actions     []string `yaml:"actions" json:"actions"`
}

type DockerContainer struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
	} `json:"State"`
}

type Service struct {
	ServiceConfig
	ContainerID string `json:"containerId"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}

type Server struct {
	services map[string]ServiceConfig
	docker   *http.Client
	static   string
}

func main() {
	services, err := loadServices("services")
	if err != nil {
		log.Fatal(err)
	}

	socket := os.Getenv("DOCKER_SOCKET")
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socket)
		},
	}}

	server := &Server{services: services, docker: client, static: "frontend/dist"}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/services", server.handleServices)
	mux.HandleFunc("/api/services/", server.handleService)
	mux.HandleFunc("/", server.handleFrontend)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("dashboard listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(mux)))
}

func loadServices(dir string) (map[string]ServiceConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read service configs: %w", err)
	}
	services := make(map[string]ServiceConfig)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var config ServiceConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if config.Name == "" {
			config.Name = strings.TrimSuffix(entry.Name(), ".yml")
		}
		if config.Container == "" {
			config.Container = config.Name
		}
		services[config.Name] = config
	}
	return services, nil
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/services" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	result := make([]Service, 0, len(s.services))
	for _, config := range s.services {
		service, err := s.serviceStatus(config)
		if err != nil {
			http.Error(w, "docker status unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}
		result = append(result, service)
	}
	writeJSON(w, result)
}

func (s *Server) handleService(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "services" {
		http.NotFound(w, r)
		return
	}
	config, ok := s.services[parts[2]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		service, err := s.serviceStatus(config)
		if err != nil {
			http.Error(w, "docker status unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, service)
		return
	}
	if len(parts) == 4 && parts[3] == "logs" && r.Method == http.MethodGet {
		s.proxyDocker(w, r, "/containers/"+config.Container+"/logs?stdout=true&stderr=true&tail=200")
		return
	}
	if len(parts) == 4 && parts[3] == "action" && r.Method == http.MethodPost {
		var body struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
		if body.Action != "start" && body.Action != "stop" && body.Action != "restart" {
			http.Error(w, "unsupported action", http.StatusBadRequest)
			return
		}
		status, err := s.serviceStatus(config)
		if err != nil {
			http.Error(w, "docker status unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}
		if body.Action == "start" && status.Running || body.Action == "stop" && !status.Running {
			http.Error(w, "action does not apply to current state", http.StatusConflict)
			return
		}
		endpoint := "/containers/" + config.Container + "/" + body.Action
		if body.Action == "restart" {
			endpoint = "/containers/" + config.Container + "/restart?t=5"
		}
		s.proxyDocker(w, r, endpoint)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) serviceStatus(config ServiceConfig) (Service, error) {
	var container DockerContainer
	req, err := http.NewRequest(http.MethodGet, "http://docker/containers/"+config.Container+"/json", nil)
	if err != nil {
		return Service{}, err
	}
	resp, err := s.docker.Do(req)
	if err != nil {
		return Service{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Service{ServiceConfig: config, Status: "not_found"}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Service{}, fmt.Errorf("container %q returned HTTP %d", config.Container, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return Service{}, fmt.Errorf("decode container status: %w", err)
	}
	return Service{ServiceConfig: config, ContainerID: container.ID, Status: container.State.Status, Running: container.State.Running}, nil
}

func (s *Server) proxyDocker(w http.ResponseWriter, r *http.Request, endpoint string) {
	req, err := http.NewRequest(r.Method, "http://docker"+endpoint, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := s.docker.Do(req)
	if err != nil {
		http.Error(w, "docker socket unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.static, filepath.Clean("/"+r.URL.Path))
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		path = filepath.Join(s.static, "index.html")
	}
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
