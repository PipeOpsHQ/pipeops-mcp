FROM golang:1.26.5-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Pre-create OAuth SQLite directories for the distroless nonroot user (UID/GID 65532).
# Default path lives under /home/nonroot so it is not wiped when a PVC is mounted at /data.
# /data/oauth is still prepared for deployments that mount a volume with fsGroup 65532.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pipeops-mcp-server ./cmd/pipeops-mcp-server && \
    mkdir -p /out/home/nonroot/.pipeops-mcp/oauth /out/data/oauth && \
    chmod 0700 /out/home/nonroot /out/home/nonroot/.pipeops-mcp /out/home/nonroot/.pipeops-mcp/oauth \
               /out/data /out/data/oauth && \
    touch /out/home/nonroot/.pipeops-mcp/oauth/.keep /out/data/oauth/.keep && \
    chmod 0600 /out/home/nonroot/.pipeops-mcp/oauth/.keep /out/data/oauth/.keep && \
    chown -R 65532:65532 /out/home/nonroot /out/data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/pipeops-mcp-server /pipeops-mcp-server
COPY --from=build --chown=65532:65532 /out/home/nonroot /home/nonroot
COPY --from=build --chown=65532:65532 /out/data /data

ENV PIPEOPS_TRANSPORT=http \
    PIPEOPS_HTTP_ADDR=:8080 \
    PIPEOPS_OAUTH_SQLITE_PATH=/home/nonroot/.pipeops-mcp/oauth/pipeops-mcp-oauth.db \
    HOME=/home/nonroot
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/pipeops-mcp-server"]
