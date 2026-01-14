package e2e_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// E2EFixture manages the environment for E2E tests.
type E2EFixture struct {
	t          *testing.T
	clientset  *kubernetes.Clientset
	namespace  string
	serverURL  string
	kubeconfig string
}

func NewE2EFixture(t *testing.T) *E2EFixture {
	return &E2EFixture{t: t}
}

func (f *E2EFixture) Client() *kubernetes.Clientset {
	return f.clientset
}

func (f *E2EFixture) Namespace() string {
	return f.namespace
}

func (f *E2EFixture) ServerURL() string {
	return f.serverURL
}

func (f *E2EFixture) SetUp(ctx context.Context) error {
	f.t.Log("Setting up E2E Fixture...")

	// 1. Environment Verification
	f.kubeconfig = os.Getenv("KUBECONFIG")
	if f.kubeconfig == "" {
		f.t.Skip("KUBECONFIG not set, skipping E2E test")
	}

	f.serverURL = os.Getenv("E2E_SERVER_URL")
	if f.serverURL == "" {
		f.serverURL = "http://localhost:8080"
	}

	requiredEnv := []string{"ANTHROPIC_API_KEY", "GITHUB_TOKEN"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			return fmt.Errorf("required environment variable %s is not set", env)
		}
	}

	// 2. Setup Kubernetes Client
	decodedKubeConfig, err := base64.StdEncoding.DecodeString(f.kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(decodedKubeConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	f.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// 3. Set Namespace
	f.namespace = "agents"
	f.t.Logf("Using namespace: %s", f.namespace)
	f.t.Logf("Using Server URL: %s", f.serverURL)

	// 4. Wait for Health
	if !waitForServer(ctx, f.ServerURL()) {
		return fmt.Errorf("server at %s failed to become healthy", f.ServerURL())
	}
	f.t.Log("Server is up and running")

	return nil
}

func (f *E2EFixture) TearDown() {
	f.t.Log("Tearing down E2E Fixture...")
}

func waitForServer(ctx context.Context, url string) bool {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
			resp, err := http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return true
			}
		}
	}
}
