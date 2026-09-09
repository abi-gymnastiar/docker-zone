package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"dashboard/internal/auth"
	"dashboard/internal/docker"
	"dashboard/internal/domain"
	"dashboard/internal/httpapi"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "/data/dashboard.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		log.Fatal(err)
	}
	repo, err := auth.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminUsername == "" || adminPassword == "" {
		log.Fatal("ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}
	if err := repo.Bootstrap(adminUsername, adminPassword); err != nil {
		log.Fatal(err)
	}

	socket := os.Getenv("DOCKER_SOCKET")
	if socket == "" {
		socket = "/var/run/docker.sock"
	}

	dockerClient := docker.NewClient(socket)
	authUseCase := auth.NewUseCase(repo)
	if discovered, err := dockerClient.Discover(); err != nil {
		log.Printf("docker discovery skipped: %v", err)
	} else if err := authUseCase.SyncServices(discovered); err != nil {
		log.Fatal(err)
	}
	storedServices, err := authUseCase.ListServices()
	if err != nil {
		log.Fatal(err)
	}
	services := make(map[string]domain.ServiceConfig, len(storedServices))
	for _, service := range storedServices {
		services[service.Name] = service
	}
	server := httpapi.NewServer(services, dockerClient, authUseCase, "frontend/dist")
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("dashboard listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
