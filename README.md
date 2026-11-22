# Larder

> A larder is a cool place (or small room) in a home where food is
> stored, usually before it's used. In the past, it was especially
> important for keeping perishable foods like meat and dairy cold before
> refrigeration became common.

A small self-hosted service to store and query recipes.

## Technical overview

- Language: Go (see `go.mod` for the required version)
- HTTP: standard library `net/http` with Go 1.22+ method/path routing
- Pure Go, no CGO (`CGO_ENABLED=0`), so it cross-builds trivially
- Interface: JSON over a REST API
- Target: Docker on a Raspberry Pi 4 (`linux/arm64`), exposed through a
  reverse proxy (caddy) rather than a published host port

## Configuration

Configuration is done through environment variables only; no config files
are needed.

| Variable | Default | Description          |
|----------|---------|----------------------|
| `PORT`   | `8080`  | TCP port to listen on |

## Docker

The service ships as a multi-stage image (Go build stage, alpine runtime)
and runs as the unprivileged user `larder` (uid 10001). Compose does not
publish a port; the container joins an existing external network so the
reverse proxy can reach it.

First start:

```sh
cp .env.example .env                    # adjust values
mkdir -p data && sudo chown 10001:10001 data
docker network create proxy             # only if it does not exist yet
docker compose up -d --build
docker compose ps                       # wait for "healthy"
```

Compose variables (`.env`):

| Variable         | Default   | Description                          |
|------------------|-----------|--------------------------------------|
| `LARDER_VERSION` | `latest`  | Image tag                            |
| `LARDER_PORT`    | `8080`    | Port inside the container            |
| `LARDER_DATA`    | `./data`  | Host directory mounted at `/data`    |
| `TZ`             | `UTC`     | Container time zone                  |
| `PROXY_NETWORK`  | `proxy`   | Name of the external network         |

Reverse proxy (caddy) block, on the same network:

```
larder.example.lan {
	reverse_proxy larder:8080
}
```

Backup: copy `data/larder.db`. Stop the container first (or use
`sqlite3 data/larder.db ".backup out.db"`) so the copy is consistent.

## API

| Method | Path       | Description                                 |
|--------|------------|---------------------------------------------|
| `GET`  | `/healthz` | Health check, returns `200 OK` (empty body) |

## Project layout

```
cmd/larder/main.go   entrypoint: reads config, registers routes, serves
```

## Development

Run the service locally:

```sh
go run ./cmd/larder
curl -i localhost:8080/healthz
```

Checks to run before committing:

```sh
gofmt -l .          # must print nothing
go vet ./...
go test ./...
```

Cross-build for the target platform:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/larder
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for
commit conventions, branch naming, and pull request guidelines.
