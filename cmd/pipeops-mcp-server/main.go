package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PipeOpsHQ/pipeops-mcp/internal/mcp"
)

func main() {
	if err := run(); err != nil {
		log.New(os.Stderr, "", 0).Printf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	transport := strings.ToLower(strings.TrimSpace(os.Getenv("PIPEOPS_TRANSPORT")))
	switch transport {
	case "", "stdio":
		server, err := mcp.NewServer()
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}
		return server.Run(ctx)
	case "http":
		return runHTTP(ctx)
	default:
		return fmt.Errorf("unsupported PIPEOPS_TRANSPORT %q: use stdio or http", transport)
	}
}

func runHTTP(ctx context.Context) error {
	config := mcp.LoadHTTPConfigFromEnv()
	handler, err := mcp.NewHTTPHandler(config)
	if err != nil {
		return fmt.Errorf("configure HTTP transport: %w", err)
	}

	server := &http.Server{
		Addr:              config.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("PipeOps MCP Streamable HTTP server listening on %s", config.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	<-shutdownDone
	return nil
}
