FROM golang:1.26.5-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p -m 0700 /out/data/oauth && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pipeops-mcp-server ./cmd/pipeops-mcp-server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/pipeops-mcp-server /pipeops-mcp-server
COPY --from=build --chown=65532:65532 /out/data /data
ENV PIPEOPS_TRANSPORT=http \
    PIPEOPS_HTTP_ADDR=:8080 \
    PIPEOPS_OAUTH_SQLITE_PATH=/data/oauth/pipeops-mcp-oauth.db
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/pipeops-mcp-server"]
