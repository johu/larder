FROM --platform=$BUILDPLATFORM golang:1.27 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/larder ./cmd/larder

FROM alpine:3.24
RUN apk add --no-cache tzdata \
    && adduser -D -u 10001 larder \
    && mkdir /data && chown larder:larder /data
COPY --from=build /out/larder /usr/local/bin/larder

ENV PORT=8080 LARDER_DB_PATH=/data/larder.db
USER larder
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:${PORT}/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/larder"]
