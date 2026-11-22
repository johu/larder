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

First start (pulls the published image):

```sh
cp .env.example .env                    # adjust values
mkdir -p data && sudo chown 10001:10001 data
docker network create proxy             # only if it does not exist yet
docker compose pull
docker compose up -d
docker compose ps                       # wait for "healthy"
```

To build the image locally instead, override the image name so the
local build is not tagged as the registry image:

```sh
LARDER_IMAGE=larder docker compose up -d --build
```

Compose variables (`.env`):

| Variable         | Default   | Description                          |
|------------------|-----------|--------------------------------------|
| `LARDER_IMAGE`   | `larder`  | Image name, e.g. `ghcr.io/johu/larder` |
| `LARDER_VERSION` | `latest`  | Image tag                            |
| `LARDER_PORT`    | `8080`    | Port inside the container            |
| `LARDER_DATA`    | `./data`  | Host directory mounted at `/data`    |
| `TZ`             | `UTC`     | Container time zone                  |
| `PROXY_NETWORK`  | `proxy`   | Name of the external network         |

### Releasing

Images are built for `linux/arm64` by the `release` GitHub Actions
workflow and published to `ghcr.io/johu/larder`. Pushing a version tag
(without a `v` prefix) triggers it, from an up-to-date `main`:

```sh
git switch main && git pull
git tag 0.1.0
git push origin 0.1.0
```

This publishes the tags `0.1.0` and `0.1`. `latest` is only moved for
stable versions, not for pre-releases such as `0.2.0-rc1`.

New packages are private by default. Either make the package public
under the package settings on GitHub, or log in on the host with a
personal access token that only has `read:packages`:

```sh
echo "$TOKEN" | docker login ghcr.io -u <github-user> --password-stdin
```

### Deploying

Set the image and a pinned version in `.env` on the host, then update:

```sh
LARDER_IMAGE=ghcr.io/johu/larder
LARDER_VERSION=0.1.0
```

```sh
docker compose pull
docker compose up -d
```

To roll back, set `LARDER_VERSION` to the previous tag and run
`docker compose up -d` again.

### Reverse proxy

Caddy block, on the same network:

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
