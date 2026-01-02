package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/compilercomplied/agent-orchestrator/api/v1"
)

func TestIntegration_FileCreation(t *testing.T) {
	// Check if claude is available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not found in PATH")
	}

	// Build the binary
	binPath := filepath.Join(t.TempDir(), "agent-orchestrator")
	// Build the main package from the module root using the module path
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/compilercomplied/agent-orchestrator")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, out)
	}

	// Get a free port
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// Setup working directory
	workingDir := t.TempDir()
	testFilePath := filepath.Join(workingDir, "test.md")

	// Run the server
	cmd := exec.Command(binPath,
		"-port", fmt.Sprintf("%d", port),
		"-working-dir", workingDir,
		"-task-timeout", "5m",
	)
	
	// Capture output for debugging
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			// Give it a moment to shut down gracefully
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				cmd.Process.Kill()
			}
		}
		if t.Failed() {
			t.Logf("Server Stdout:\n%s", stdout.String())
			t.Logf("Server Stderr:\n%s", stderr.String())
		}
	}()

	// Wait for health check
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if !waitForServer(baseURL) {
		t.Logf("Server Stdout:\n%s", stdout.String())
		t.Logf("Server Stderr:\n%s", stderr.String())
		t.Fatal("Server failed to become healthy")
	}

	// Prepare request
	taskRequest := v1.TaskRequest{
		Task: "write hello world in a file named test.md",
	}
	requestBody, err := json.Marshal(taskRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Send HTTP request
	url := fmt.Sprintf("%s/api/tasks", baseURL)
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
	var taskResponse v1.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if taskResponse.Status != "accepted" {
		t.Fatalf("Expected status 'accepted', got '%s'", taskResponse.Status)
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

func waitForServer(baseURL string) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return true
			}
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
