FROM golang:1.26.5-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Pre-create OAuth SQLite directory under /data (durable when a PVC is mounted
# at /data). Run as root so root-owned PVCs work without fsGroup/UID gymnastics.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pipeops-mcp-server ./cmd/pipeops-mcp-server && \
    mkdir -p /out/data/oauth && \
    chmod 0700 /out/data /out/data/oauth

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/pipeops-mcp-server /pipeops-mcp-server
COPY --from=build /out/data /data

ENV PIPEOPS_TRANSPORT=http \
    PIPEOPS_HTTP_ADDR=:8080 \
    PIPEOPS_OAUTH_SQLITE_PATH=/data/oauth/pipeops-mcp-oauth.db
EXPOSE 8080
# Root: matches typical PVC ownership so OAuth SQLite survives deploys without
# fsGroup:65532. Prefer network policy + private volume over container UID.
ENTRYPOINT ["/pipeops-mcp-server"]
