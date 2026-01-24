package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	v1 "github.com/compilercomplied/agent-orchestrator/api/v1"
	"github.com/compilercomplied/agent-orchestrator/internal/agent"
	"github.com/compilercomplied/agent-orchestrator/internal/logging"
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
		logging.Printf("Failed to decode request: %v", err)
		h.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Task) == "" {
		h.sendError(w, "task field is required and cannot be empty", http.StatusBadRequest)
		return
	}

	logging.Printf("Received task request: %s", req.Task)

	podName, err := h.agentManager.CreateTask(r.Context(), req.Task)
	if err != nil {
		logging.Printf("Failed to create task: %v", err)
		h.sendError(w, "failed to create task", http.StatusInternalServerError)
		return
	}

	go func() {
		// Use background context instead of request context to avoid cancellation
		if err := h.agentManager.WatchTask(context.Background(), podName); err != nil {
			logging.Printf("Task execution failed for pod %s: %v", podName, err)
		}
	}()

	response := v1.TaskResponse{
		Status:  "accepted",
		Message: "Task has been accepted and is being processed",
		PodName: podName,
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
