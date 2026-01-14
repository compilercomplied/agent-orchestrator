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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	t.Log("Waiting for Pod execution...")

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for Pod completion")
		case <-ticker.C:
			pods, err := fixture.Client().CoreV1().Pods(fixture.Namespace()).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Logf("Error listing pods: %v", err)
				continue
			}

			if len(pods.Items) == 0 {
				continue
			}

			for _, pod := range pods.Items {
				t.Logf("Pod %s is in phase: %s", pod.Name, pod.Status.Phase)
				if pod.Status.Phase == corev1.PodSucceeded {
					t.Logf("Pod %s succeeded!", pod.Name)

					logReq := fixture.Client().CoreV1().Pods(fixture.Namespace()).GetLogs(pod.Name, &corev1.PodLogOptions{})
					logs, err := logReq.Do(ctx).Raw()
					if err == nil {
						t.Logf("Pod Logs:\n%s", string(logs))
					}
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
}