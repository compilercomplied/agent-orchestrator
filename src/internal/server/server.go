package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/compilercomplied/agent-orchestrator/src/internal/agent"
	"github.com/compilercomplied/agent-orchestrator/src/internal/configuration"
	"github.com/compilercomplied/agent-orchestrator/src/internal/handler"
	"github.com/compilercomplied/agent-orchestrator/src/internal/logging"
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

func Run() {
	logging.Init(configuration.GetEnv("LOG_FORMAT"))

	serverCfg, agentCfg := configuration.Load()

	agentManager, err := agent.NewManager(serverCfg.KubeConfig, serverCfg.Namespace, serverCfg.TaskTimeout, agentCfg)
	if err != nil {
		logging.Fatalf("Failed to initialize agent manager: %v", err)
	}

	if err := agentManager.ValidateConfig(); err != nil {
		logging.Fatalf("Agent manager validation failed: %v", err)
	}

	taskHandler := handler.NewTaskHandler(agentManager)

	serverConfig := Config{
		Port:         serverCfg.Port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srv := NewServer(serverConfig, taskHandler)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logging.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logging.Fatalf("Server forced to shutdown: %v", err)
	}

	logging.Printf("Server exited")
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

	logging.Printf("Starting server on port %d", s.config.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	logging.Printf("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		logging.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		logging.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}
