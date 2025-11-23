package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/compilercomplied/agent-orchestrator/internal/agent"
	"github.com/compilercomplied/agent-orchestrator/internal/handler"
	"github.com/compilercomplied/agent-orchestrator/internal/server"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	claudeBinary := flag.String("claude-binary", "claude", "Path to claude binary")
	workingDir := flag.String("working-dir", "/tmp/agent-tasks", "Working directory for agent execution")
	taskTimeout := flag.Duration("task-timeout", 30*time.Minute, "Timeout for task execution")
	flag.Parse()

	if err := os.MkdirAll(*workingDir, 0755); err != nil {
		log.Fatalf("Failed to create working directory: %v", err)
	}

	agentManager := agent.NewManager(*claudeBinary, *workingDir, *taskTimeout)
	if err := agentManager.ValidateClaudeBinary(); err != nil {
		log.Fatalf("Claude binary validation failed: %v", err)
	}

	taskHandler := handler.NewTaskHandler(agentManager)

	serverConfig := server.Config{
		Port:         *port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srv := server.NewServer(serverConfig, taskHandler)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
