package httpapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	mux.HandleFunc("/api/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/admin/groups", s.handleAdminGroups)
	mux.HandleFunc("/api/admin/services", s.handleAdminServices)
	mux.HandleFunc("/api/admin/sync", s.handleAdminSync)
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
	search := strings.ToLower(r.URL.Query().Get("search"))
	result := make([]domain.Service, 0, len(s.services))
	for _, config := range s.services {
		if !config.Enabled && user.Role != "admin" {
			continue
		}
		groups, err := s.groupsFor(config)
		if err != nil {
			http.Error(w, "service groups unavailable", http.StatusInternalServerError)
			return
		}
		allowed, err := s.auth.Allowed(user, groups, "services:view")
		if err != nil {
			http.Error(w, "authorization unavailable", http.StatusInternalServerError)
			return
		}
		if !allowed {
			continue
		}
		if search != "" && !matchesService(config, search) {
			continue
		}
		service, err := s.serviceStatus(config)
		if err != nil {
			http.Error(w, "docker status unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}
		result = append(result, service)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	writeJSON(w, paginate(result, r))
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
	groups, err := s.groupsFor(config)
	if err != nil {
		http.Error(w, "service groups unavailable", http.StatusInternalServerError)
		return false
	}
	allowed, err := s.auth.Allowed(user, groups, permission)
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

func (s *Server) groupsFor(config domain.ServiceConfig) ([]string, error) {
	return s.auth.ServiceGroups(config.Name, config.Groups)
}

func (s *Server) adminUser(r *http.Request) (auth.User, bool) {
	user, err := s.auth.Current(r)
	return user, err == nil && user.Role == "admin"
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(r)
	if !ok {
		http.Error(w, "admin access required", http.StatusForbidden)
		return
	}
	_ = user
	switch r.Method {
	case http.MethodGet:
		users, err := s.auth.ListUsers()
		if err != nil {
			http.Error(w, "could not list users", http.StatusInternalServerError)
			return
		}
		writeJSON(w, users)
	case http.MethodPost:
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Username == "" || input.Password == "" {
			http.Error(w, "username and password are required", http.StatusBadRequest)
			return
		}
		if input.Role == "" {
			input.Role = "viewer"
		}
		if err := s.auth.CreateUser(input.Username, input.Password, input.Role); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminUser(r); !ok {
		http.Error(w, "admin access required", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		groups, err := s.auth.ListGroups()
		if err != nil {
			http.Error(w, "could not list groups", http.StatusInternalServerError)
			return
		}
		writeJSON(w, groups)
	case http.MethodPost:
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Name == "" {
			http.Error(w, "group name is required", http.StatusBadRequest)
			return
		}
		if err := s.auth.CreateGroup(input.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		var input struct {
			GroupID int64  `json:"groupId"`
			UserID  int64  `json:"userId"`
			Role    string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid membership", http.StatusBadRequest)
			return
		}
		if err := s.auth.SetGroupMember(input.GroupID, input.UserID, input.Role); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminServices(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminUser(r); !ok {
		http.Error(w, "admin access required", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		result := make([]auth.ServiceGroups, 0, len(s.services))
		for _, service := range s.services {
			search := strings.ToLower(r.URL.Query().Get("search"))
			if search != "" && !matchesService(service, search) {
				continue
			}
			groups, err := s.groupsFor(service)
			if err != nil {
				http.Error(w, "could not list service groups", http.StatusInternalServerError)
				return
			}
			result = append(result, auth.ServiceGroups{Name: service.Name, Container: service.Container, ContainerID: service.ContainerID, Description: service.Description, Actions: service.Actions, Groups: groups, Enabled: service.Enabled, Orphaned: service.Orphaned})
		}
		sort.Slice(result, func(i, j int) bool {
			return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		})
		writeJSON(w, paginate(result, r))
	case http.MethodPut:
		var input struct {
			Name        string   `json:"name"`
			Groups      []string `json:"groups"`
			Description string   `json:"description"`
			Enabled     bool     `json:"enabled"`
			Actions     []string `json:"actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || s.services[input.Name].Name == "" {
			http.Error(w, "invalid service groups", http.StatusBadRequest)
			return
		}
		if err := s.auth.SetServiceGroups(input.Name, input.Groups); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.auth.SetServiceMetadata(input.Name, input.Description, input.Enabled, input.Actions); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if _, ok := s.adminUser(r); !ok {
		http.Error(w, "admin access required", http.StatusForbidden)
		return
	}
	if err := s.syncServices(); err != nil {
		http.Error(w, "docker sync failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) syncServices() error {
	discovered, err := s.docker.Discover()
	if err != nil {
		return err
	}
	if err := s.auth.SyncServices(discovered); err != nil {
		return err
	}
	services, err := s.auth.ListServices()
	if err != nil {
		return err
	}
	s.services = make(map[string]domain.ServiceConfig, len(services))
	for _, service := range services {
		s.services[service.Name] = service
	}
	return nil
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, "not authenticated", http.StatusUnauthorized)
}

func matchesService(service domain.ServiceConfig, search string) bool {
	return strings.Contains(strings.ToLower(service.Name), search) ||
		strings.Contains(strings.ToLower(service.Container), search) ||
		strings.Contains(strings.ToLower(service.Description), search) ||
		strings.Contains(strings.ToLower(service.ContainerID), search)
}

func paginate(items any, r *http.Request) map[string]any {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 5)
	if page < 1 {
		page = 1
	}
	if pageSize != 5 && pageSize != 10 && pageSize != 20 && pageSize != 50 {
		pageSize = 5
	}
	switch values := items.(type) {
	case []domain.Service:
		return pageResult(values, page, pageSize)
	case []auth.ServiceGroups:
		return pageResult(values, page, pageSize)
	default:
		return map[string]any{"items": []any{}, "page": page, "pageSize": pageSize, "total": 0}
	}
}

func pageResult[T any](items []T, page, pageSize int) map[string]any {
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return map[string]any{"items": items[start:end], "page": page, "pageSize": pageSize, "total": len(items)}
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
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
