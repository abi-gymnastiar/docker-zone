# My Docker Zone

A small, deliberately old-school dashboard for Docker services. Service metadata and
available actions live in individual YAML files under [`services/`](./services/).

The backend is organized into:

- `cmd/dashboard`: application entrypoint and environment wiring.
- `internal/config`: YAML service configuration loading.
- `internal/domain`: service entities shared by the application layers.
- `internal/docker`: Docker socket client/repository.
- `internal/httpapi`: REST handlers and static frontend serving.

The frontend keeps pages, reusable components, and their component styles in
separate files under [`frontend/src/`](./frontend/src/).

Images and GIFs for the optional evil overlay belong in
[`frontend/media/`](./frontend/media/). The frontend selects three of them at
random each time the page loads and places them around the corners of the
transparent overlay.

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
- `ADMIN_USERNAME` and `ADMIN_PASSWORD` bootstrap the first administrator.
- `DATABASE_DIR` sets the host directory containing the SQLite database.
- `DATABASE_PATH` sets the SQLite database path inside the container.

The first administrator is created only when the database has no users. Change
the example password before deploying. Authentication uses server-side sessions
in secure HTTP-only cookies. Services are protected by group membership and
role permissions; the Minecraft service belongs to the `minecraft` group.

SQLite is an embedded database, not a network database server, so there is no
database port to expose. The database is stored in `DATABASE_DIR` on the host.
To inspect it with DBeaver, stop the dashboard first, copy the file to your
workstation over SSH, and open the copy as a SQLite database:

```sh
docker compose stop dashboard
scp your-server:/path/to/project/data/dashboard.db ./dashboard.db
docker compose start dashboard
```

Do not edit a live SQLite file from another machine. For a live remote database
connection, migrate to PostgreSQL later or add a deliberately authenticated
database administration service instead of exposing the SQLite file.

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
   export ADMIN_USERNAME=admin
   export ADMIN_PASSWORD=change-this-password
   go run ./cmd/dashboard
   ```

Open <http://localhost:8080>. The process needs permission to access
`/var/run/docker.sock`. Set `DOCKER_SOCKET` or `LISTEN_ADDR` to override the defaults.

The development frontend can run with `npm run dev` in `frontend/`; its Vite proxy
forwards `/api` requests to the Go server.
