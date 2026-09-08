package httpapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dashboard/internal/auth"
	"dashboard/internal/docker"
	"dashboard/internal/domain"
)

type Server struct {
	services map[string]domain.ServiceConfig
	docker   *docker.Client
	auth     *auth.UseCase
	static   string
}

func NewServer(services map[string]domain.ServiceConfig, dockerClient *docker.Client, authUseCase *auth.UseCase, static string) *Server {
	return &Server{services: services, docker: dockerClient, auth: authUseCase, static: static}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/services", s.handleServices)
	mux.HandleFunc("/api/services/", s.handleService)
	mux.HandleFunc("/", s.handleFrontend)
	return logging(mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "invalid login", http.StatusBadRequest)
		return
	}
	user, err := s.auth.Login(w, credentials.Username, credentials.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	writeJSON(w, user)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if err := s.auth.Logout(w, r); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	user, err := s.auth.Current(r)
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	writeJSON(w, user)
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/services" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	user, err := s.auth.Current(r)
	if err != nil {
		unauthorized(w)
		return
	}
	result := make([]domain.Service, 0, len(s.services))
	for _, config := range s.services {
		allowed, err := s.auth.Allowed(user, config.Groups, "services:view")
		if err != nil {
			http.Error(w, "authorization unavailable", http.StatusInternalServerError)
			return
		}
		if !allowed {
			continue
		}
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
	user, err := s.auth.Current(r)
	if err != nil {
		unauthorized(w)
		return
	}

	switch {
	case len(parts) == 3 && r.Method == http.MethodGet:
		if !s.requirePermission(w, user, config, "services:view") {
			return
		}
		s.handleServiceStatus(w, config)
	case len(parts) == 4 && parts[3] == "logs" && r.Method == http.MethodGet:
		if !s.requirePermission(w, user, config, "services:logs") {
			return
		}
		s.proxyDocker(w, r, "/containers/"+config.Container+"/logs?stdout=true&stderr=true&tail=200")
	case len(parts) == 4 && parts[3] == "action" && r.Method == http.MethodPost:
		s.handleAction(w, r, config, user)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleServiceStatus(w http.ResponseWriter, config domain.ServiceConfig) {
	service, err := s.serviceStatus(config)
	if err != nil {
		http.Error(w, "docker status unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, service)
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request, config domain.ServiceConfig, user auth.User) {
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
	if !s.requirePermission(w, user, config, "services:"+body.Action) {
		return
	}

	status, err := s.docker.Status(config.Container)
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
}

func (s *Server) requirePermission(w http.ResponseWriter, user auth.User, config domain.ServiceConfig, permission string) bool {
	allowed, err := s.auth.Allowed(user, config.Groups, permission)
	if err != nil {
		http.Error(w, "authorization unavailable", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, "not authenticated", http.StatusUnauthorized)
}

func (s *Server) serviceStatus(config domain.ServiceConfig) (domain.Service, error) {
	status, err := s.docker.Status(config.Container)
	if err != nil {
		return domain.Service{}, err
	}
	return domain.Service{
		ServiceConfig: config,
		ContainerID:   status.ID,
		Status:        status.State,
		Running:       status.Running,
	}, nil
}

func (s *Server) proxyDocker(w http.ResponseWriter, r *http.Request, endpoint string) {
	resp, err := s.docker.Proxy(r.Method, endpoint, r.Body)
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
