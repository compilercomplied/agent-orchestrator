package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	v1 "github.com/compilercomplied/agent-orchestrator/api/v1"
	"github.com/compilercomplied/agent-orchestrator/internal/agent"
)

type TaskHandler struct {
	agentManager *agent.Manager
}

func NewTaskHandler(agentManager *agent.Manager) *TaskHandler {
	return &TaskHandler{
		agentManager: agentManager,
	}
}

func (h *TaskHandler) HandleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req v1.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		h.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Task) == "" {
		h.sendError(w, "task field is required and cannot be empty", http.StatusBadRequest)
		return
	}

	log.Printf("Received task request: %s", req.Task)

	go func() {
		// Use background context instead of request context to avoid cancellation
		if err := h.agentManager.ExecuteTask(context.Background(), req.Task); err != nil {
			log.Printf("Task execution failed: %v", err)
		}
	}()

	response := v1.TaskResponse{
		Status:  "accepted",
		Message: "Task has been accepted and is being processed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (h *TaskHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v1.ErrorResponse{Error: message})
}
