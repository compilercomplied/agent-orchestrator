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

// GetEnv retrieves the value of the environment variable named by the key.
// Panics if the variable is not present.
func GetEnv(key string) string {
	value, ok := os.LookupEnv(ENV_PREFIX + key)
	if !ok {
		panic(fmt.Sprintf("environment variable %s%s is required", ENV_PREFIX, key))
	}
	return value
}

func Load() *ServerConfig {
	serverConfig := &ServerConfig{
		Port:        8080,
		KubeConfig:  GetEnv("KUBECONFIG"),
		Namespace:   "agents",
		TaskTimeout: 30 * time.Minute,
	}

	return serverConfig
}
