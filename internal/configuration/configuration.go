package configuration

import (
	"fmt"
	"os"
	"time"
)

type ServerConfig struct {
	Port        int
	KubeConfig  string
	Namespace   string
	TaskTimeout time.Duration
}

type AgentConfig struct {
	GithubToken  string
	AnthropicKey string
}

// GetEnv retrieves the value of the environment variable named by the key.
// If the variable is not present, it returns an error.
func GetEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s is required", key)
	}
	return value, nil
}

func Load() (*ServerConfig, *AgentConfig, error) {
	kubeConfig, err := GetEnv("KUBECONFIG")
	if err != nil {
		return nil, nil, err
	}

	serverConfig := &ServerConfig{
		Port:        8080,
		KubeConfig:  kubeConfig,
		Namespace:   "agents",
		TaskTimeout: 30 * time.Minute,
	}

	githubToken, err := GetEnv("GITHUB_TOKEN")
	if err != nil {
		return nil, nil, err
	}

	anthropicKey, err := GetEnv("ANTHROPIC_API_KEY")
	if err != nil {
		return nil, nil, err
	}

	agentConfig := &AgentConfig{
		GithubToken:  githubToken,
		AnthropicKey: anthropicKey,
	}

	return serverConfig, agentConfig, nil
}
