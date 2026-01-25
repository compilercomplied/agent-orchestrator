package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	v1 "github.com/compilercomplied/agent-orchestrator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestE2E_TaskExecution(t *testing.T) {
	fixture := NewE2EFixture(t)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := fixture.SetUp(ctx); err != nil {
		t.Fatalf("Fixture setup failed: %v", err)
	}
	defer fixture.TearDown()

	t.Log("Sending task request...")
	taskPayload := v1.TaskRequest{
		Task: "Please echo 'E2E_SUCCESS' to stdout. This is a connectivity test.",
	}
	body, _ := json.Marshal(taskPayload)

	req, err := http.NewRequestWithContext(ctx, "POST", fixture.ServerURL()+"/api/tasks", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send task request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted, got %d", resp.StatusCode)
	}
	t.Log("Task accepted by API")

	var taskResp v1.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if taskResp.PodName == "" {
		t.Fatal("Response did not contain PodName")
	}
	t.Logf("Tracking Pod: %s", taskResp.PodName)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	t.Log("Waiting for Pod execution...")

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for Pod completion")
		case <-ticker.C:
			pod, err := fixture.Client().CoreV1().Pods(fixture.Namespace()).Get(ctx, taskResp.PodName, metav1.GetOptions{})
			if err != nil {
				t.Logf("Error getting pod %s: %v", taskResp.PodName, err)
				continue
			}

			// Verify Labels for Observability
			if pod.Labels["app"] != "claude-worker" {
				t.Errorf("Pod %s missing expected label 'app: claude-worker'. Got: %v", pod.Name, pod.Labels["app"])
			}
			if pod.Labels["app.kubernetes.io/managed-by"] != "agent-orchestrator" {
				t.Errorf("Pod %s missing expected label 'app.kubernetes.io/managed-by: agent-orchestrator'. Got: %v", pod.Name, pod.Labels["app.kubernetes.io/managed-by"])
			}

			t.Logf("Pod %s is in phase: %s", pod.Name, pod.Status.Phase)
			
			if pod.Status.Phase == corev1.PodSucceeded {
				t.Logf("Pod %s succeeded!", pod.Name)
				return
			}
			if pod.Status.Phase == corev1.PodFailed {
				logReq := fixture.Client().CoreV1().Pods(fixture.Namespace()).GetLogs(pod.Name, &corev1.PodLogOptions{})
				logs, _ := logReq.Do(ctx).Raw()
				t.Fatalf("Pod %s failed. Logs:\n%s", pod.Name, string(logs))
			}
		}
	}
}