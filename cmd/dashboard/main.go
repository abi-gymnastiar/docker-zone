package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"dashboard/internal/auth"
	"dashboard/internal/config"
	"dashboard/internal/docker"
	"dashboard/internal/httpapi"
)

func main() {
	services, err := config.LoadServices("services")
	if err != nil {
		log.Fatal(err)
	}
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
	for _, service := range services {
		if err := repo.EnsureGroups(service.Groups); err != nil {
			log.Fatal(err)
		}
	}

	socket := os.Getenv("DOCKER_SOCKET")
	if socket == "" {
		socket = "/var/run/docker.sock"
	}

	server := httpapi.NewServer(services, docker.NewClient(socket), auth.NewUseCase(repo), "frontend/dist")
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("dashboard listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
