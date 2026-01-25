package configuration

import (
	"fmt"
	"os"
	"time"
)

const ENV_PREFIX = "AO_"

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
// Panics if the variable is not present.
func GetEnv(key string) string {
	value, ok := os.LookupEnv(ENV_PREFIX + key)
	if !ok {
		panic(fmt.Sprintf("environment variable %s%s is required", ENV_PREFIX, key))
	}
	return value
}

func Load() (*ServerConfig, *AgentConfig) {
	serverConfig := &ServerConfig{
		Port:        8080,
		KubeConfig:  GetEnv("KUBECONFIG"),
		Namespace:   "agents",
		TaskTimeout: 30 * time.Minute,
	}

	agentConfig := &AgentConfig{
		GithubToken:  GetEnv("GITHUB_TOKEN"),
		AnthropicKey: GetEnv("ANTHROPIC_API_KEY"),
	}

	return serverConfig, agentConfig
}
