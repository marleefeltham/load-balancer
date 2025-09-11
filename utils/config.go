package utils

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type LBStrategy int

const (
	RoundRobin LBStrategy = iota
	LeastConnected
)

func GetLBStrategy(strategy string) LBStrategy {
	switch strategy {
	case "least-connection":
		return LeastConnected
	default:
		return RoundRobin
	}
}

type Config struct {
	Port                int      `yaml:"lb_port"`
	MaxAttemptLimit     int      `yaml:"max_attempt_limit"`
	Backends            []string `yaml:"backends"`
	Strategy            string   `yaml:"strategy"`
	HealthCheckInterval int      `yaml:"healthcheck_interval"`
	BackendTimeout      int      `yaml:"backend_timeout"`
	ShutdownTimeout     int      `yaml:"shutdown_timeout"`
}

const MAX_LB_ATTEMPTS int = 3

func GetLBConfig() (*Config, error) {
	var config Config

	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	if len(config.Backends) == 0 {
		return nil, errors.New("backend hosts expected, none provided")
	}

	if config.Port == 0 {
		return nil, errors.New("load balancer port not found")
	}

	// set health timeout if not configured
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 20 // default to 20 seconds
	}

	// set backend timeout if not configured
	if config.BackendTimeout <= 0 {
		config.BackendTimeout = 2 // default to 2 seconds
	}

	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 10 // default to 2 seconds
	}

	// set max attempt limit if not configured
	if config.MaxAttemptLimit <= 0 {
		config.MaxAttemptLimit = 3
	}

	return &config, nil
}
