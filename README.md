# My Docker Zone

A small, deliberately old-school dashboard for Docker services. Service metadata and
available actions live in individual YAML files under [`services/`](./services/).

## Run with Docker Compose

The recommended home-server deployment mounts the host Docker socket into the
dashboard container so it can inspect and manage configured containers:

```sh
cp .env.example .env
# Edit .env if needed, then:
docker compose up -d --build
```

Open <http://localhost:8080>. After pulling new changes, run the same command
again to rebuild and recreate the container. Useful commands:

```sh
docker compose logs -f
docker compose ps
docker compose down
```

Compose reads deployment settings from the untracked `.env` file. The available
settings are documented in [`.env.example`](./.env.example):

- `DASHBOARD_PORT` controls the host port.
- `LISTEN_ADDR` controls the address inside the container.
- `DOCKER_SOCKET` sets the host Docker socket path.

The Docker socket grants the dashboard broad control over the host Docker
daemon. Keep this service on a trusted network and do not expose port 8080
directly to the public internet.

## Run locally

1. Install Go 1.22+ and Node.js 18+.
2. Build the frontend:

   ```sh
   cd frontend
   npm install
   npm run build
   cd ..
   ```

3. Start the dashboard:

   ```sh
   go run .
   ```

Open <http://localhost:8080>. The process needs permission to access
`/var/run/docker.sock`. Set `DOCKER_SOCKET` or `LISTEN_ADDR` to override the defaults.

The development frontend can run with `npm run dev` in `frontend/`; its Vite proxy
forwards `/api` requests to the Go server.
