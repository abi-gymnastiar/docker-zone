package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"dashboard/internal/config"
	"dashboard/internal/docker"
	"dashboard/internal/httpapi"
	postgresrepo "dashboard/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	configs, err := config.LoadServices("services")
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(fmt.Errorf("connect to postgres: %w", err))
	}
	migration, err := os.ReadFile("migrations/001_services.sql")
	if err != nil {
		log.Fatal(fmt.Errorf("read migrations: %w", err))
	}
	if err := postgresrepo.RunMigrations(ctx, pool, string(migration)); err != nil {
		log.Fatal(err)
	}
	services := postgresrepo.NewServices(pool)
	if err := services.Sync(ctx, configs); err != nil {
		log.Fatal(fmt.Errorf("sync service configs: %w", err))
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
