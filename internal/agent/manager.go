package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Manager struct {
	clientset *kubernetes.Clientset
	namespace string
	timeout   time.Duration
	image     string
}

func NewManager(kubeconfigBase64, namespace string, timeout time.Duration) (*Manager, error) {
	if kubeconfigBase64 == "" {
		return nil, fmt.Errorf("kubeconfig base64 string is required")
	}

	decodedKubeConfig, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(decodedKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Manager{
		clientset: clientset,
		namespace: namespace,
		timeout:   timeout,
		image:     "ghcr.io/compilercomplied/claude-agent:latest",
	}, nil
}

func (m *Manager) ExecuteTask(ctx context.Context, task string) error {
	podName := m.generatePodName(task)
	log.Printf("Starting Claude Code agent in k8s. Pod: %s, Task length: %d", podName, len(task))

	// Prepare Environment Variables
	envVars := []corev1.EnvVar{
		{
			Name:  "ANTHROPIC_API_KEY",
			Value: os.Getenv("ANTHROPIC_API_KEY"),
		},
		{
			Name:  "GITHUB_TOKEN",
			Value: os.Getenv("GITHUB_TOKEN"),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "claude-agent",
					Image: m.image,
					// Entrypoint is set in Dockerfile, we provide args to it.
					// entrypoint.sh does `exec claude "$@"`
					Args: []string{"--dangerously-skip-permissions", task},
					Env:  envVars,
				},
			},
		},
	}

	// Create Pod
	_, err := m.clientset.CoreV1().Pods(m.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Cleanup defer
	defer func() {
		log.Printf("Cleaning up pod %s", podName)
		err := m.clientset.CoreV1().Pods(m.namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
		if err != nil {
			log.Printf("Failed to delete pod %s: %v", podName, err)
		}
	}()

	// Wait for Pod completion
	log.Printf("Waiting for pod %s to complete...", podName)
	
	waitCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("timeout waiting for pod execution")
		case <-ticker.C:
			p, err := m.clientset.CoreV1().Pods(m.namespace).Get(waitCtx, podName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Error getting pod status: %v", err)
				continue
			}

			switch p.Status.Phase {
			case corev1.PodSucceeded:
				log.Printf("Pod %s succeeded", podName)
				m.logPodOutput(ctx, podName)
				return nil
			case corev1.PodFailed:
				log.Printf("Pod %s failed", podName)
				m.logPodOutput(ctx, podName)
				return fmt.Errorf("pod %s failed", podName)
			case corev1.PodUnknown:
				log.Printf("Pod %s status unknown", podName)
			default:
				// Pending or Running
			}
		}
	}
}

func (m *Manager) logPodOutput(ctx context.Context, podName string) {
	req := m.clientset.CoreV1().Pods(m.namespace).GetLogs(podName, &corev1.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Printf("Error opening log stream for pod %s: %v", podName, err)
		return
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Printf("Error copying log stream for pod %s: %v", podName, err)
		return
	}

	output := buf.String()
	if output != "" {
		log.Printf("=== POD LOGS (%s) ===\n%s\n=======================", podName, output)
	} else {
		log.Printf("Pod %s produced no logs", podName)
	}
}

func (m *Manager) generatePodName(task string) string {
	// Hash the task content to get a base
	hash := sha256.Sum256([]byte(task))
	hashStr := hex.EncodeToString(hash[:])
	
	// Add random suffix to ensure uniqueness
	randBytes := make([]byte, 3)
	if _, err := rand.Read(randBytes); err != nil {
		// Fallback if rand fails (unlikely)
		return fmt.Sprintf("claude-%s-%d", hashStr[:10], time.Now().UnixNano()%1000)
	}
	randSuffix := hex.EncodeToString(randBytes)
	
	return fmt.Sprintf("claude-%s-%s", hashStr[:10], randSuffix)
}

func (m *Manager) ValidateConfig() error {
	// Simple check if we can list pods (even empty list) to verify permissions
	_, err := m.clientset.CoreV1().Pods(m.namespace).List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to validate k8s connection: %w", err)
	}
	return nil
}