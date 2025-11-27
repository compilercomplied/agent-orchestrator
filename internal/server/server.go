package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

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
