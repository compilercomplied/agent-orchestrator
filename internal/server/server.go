package server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/compilercomplied/agent-orchestrator/internal/agent"
	"github.com/compilercomplied/agent-orchestrator/internal/handler"
)

type Config struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Server struct {
	config      Config
	httpServer  *http.Server
	taskHandler *handler.TaskHandler
}

func NewServer(config Config, taskHandler *handler.TaskHandler) *Server {
	return &Server{
		config:      config,
		taskHandler: taskHandler,
	}
}

// Run parses flags, initializes dependencies, and starts the server with graceful shutdown.
func Run() {
	port := flag.Int("port", 8080, "Server port")
	kubeconfig := flag.String("kubeconfig", "", "Base64 encoded kubeconfig content")
	namespace := flag.String("namespace", "agents", "Kubernetes namespace")
	taskTimeout := flag.Duration("task-timeout", 30*time.Minute, "Timeout for task execution")
	flag.Parse()

	agentManager, err := agent.NewManager(*kubeconfig, *namespace, *taskTimeout)
	if err != nil {
		log.Fatalf("Failed to initialize agent manager: %v", err)
	}

	if err := agentManager.ValidateConfig(); err != nil {
		log.Fatalf("Agent manager validation failed: %v", err)
	}

	taskHandler := handler.NewTaskHandler(agentManager)

	serverConfig := Config{
		Port:         *port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srv := NewServer(serverConfig, taskHandler)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
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

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", s.taskHandler.HandleTask)
	mux.HandleFunc("/health", s.healthCheck)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", s.config.Port),
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	log.Printf("Starting server on port %d", s.config.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}