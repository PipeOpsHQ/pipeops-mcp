package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/PipeOpsHQ/pipeops-mcp-server/internal/mcp"
)

func main() {
	if err := run(); err != nil {
		log.New(os.Stderr, "", 0).Printf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Create MCP server
	server, err := mcp.NewServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

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

	// Run server
	return server.Run(ctx)
}
