package main

import (
	"log"
	"net/http"
	"os"

	"dashboard/internal/config"
	"dashboard/internal/docker"
	"dashboard/internal/httpapi"
)

func main() {
	services, err := config.LoadServices("services")
	if err != nil {
		log.Fatal(err)
	}

	socket := os.Getenv("DOCKER_SOCKET")
	if socket == "" {
		socket = "/var/run/docker.sock"
	}

	server := httpapi.NewServer(services, docker.NewClient(socket), "frontend/dist")
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("dashboard listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}
