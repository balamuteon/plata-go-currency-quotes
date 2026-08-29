package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func stringEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func portEnv(name string) (string, error) {
	value, err := stringEnv(name)
	if err != nil {
		return "", err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return "", fmt.Errorf("invalid %s: %w", name, err)
	}
	if parsed <= 0 || parsed > 65535 {
		return "", fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return strconv.Itoa(parsed), nil
}

func durationEnv(name string) (time.Duration, error) {
	value, err := stringEnv(name)
	if err != nil {
		return 0, err
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return parsed, nil
}

func intEnv(name string) (int, error) {
	value, err := stringEnv(name)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}
