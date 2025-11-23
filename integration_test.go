package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compilercomplied/agent-orchestrator/internal/agent"
	"github.com/compilercomplied/agent-orchestrator/internal/handler"
	"github.com/compilercomplied/agent-orchestrator/internal/server"
)

func TestIntegration_FileCreation(t *testing.T) {
	// Setup test directory
	testDir := t.TempDir()
	testFilePath := filepath.Join(testDir, "test.md")

	// Initialize agent manager
	claudeBinary := "claude"
	agentManager := agent.NewManager(claudeBinary, testDir, 5*time.Minute)

	// Validate claude binary is available
	if err := agentManager.ValidateClaudeBinary(); err != nil {
		t.Skipf("Claude binary not found, skipping integration test: %v", err)
	}

	// Initialize task handler
	taskHandler := handler.NewTaskHandler(agentManager)

	// Initialize server
	serverConfig := server.Config{
		Port:         8081, // Use different port to avoid conflicts
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srv := server.NewServer(serverConfig, taskHandler)

	// Start server in goroutine
	serverStarted := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != http.ErrServerClosed {
			serverStarted <- err
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Ensure server shuts down after test
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("Failed to shutdown server: %v", err)
		}
	}()

	// Check if server started successfully
	select {
	case err := <-serverStarted:
		t.Fatalf("Server failed to start: %v", err)
	default:
		// Server started successfully
	}

	// Prepare request
	taskRequest := map[string]string{
		"task": "write hello world in a file named test.md",
	}
	requestBody, err := json.Marshal(taskRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Send HTTP request
	url := fmt.Sprintf("http://localhost:%d/api/tasks", serverConfig.Port)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	// Decode response
	var taskResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&taskResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if taskResponse["status"] != "accepted" {
		t.Fatalf("Expected status 'accepted', got '%s'", taskResponse["status"])
	}

	t.Logf("Task accepted successfully")

	// Wait for the task to complete (poll for file existence)
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fileFound := false
	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for file to be created")
		case <-ticker.C:
			if _, err := os.Stat(testFilePath); err == nil {
				fileFound = true
				goto FileCreated
			}
		}
	}

FileCreated:
	if !fileFound {
		t.Fatal("File was not created")
	}

	t.Logf("File created at: %s", testFilePath)

	// Read file content
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Verify content
	contentStr := string(content)
	if contentStr != "hello world" && contentStr != "hello world\n" {
		t.Fatalf("Expected content 'hello world', got '%s'", contentStr)
	}

	t.Logf("File content verified successfully: %s", contentStr)
}
