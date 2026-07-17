package config

import (
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	ServerTimeout time.Duration
	WorkerCount   int
	AppPort       string
	LogLevel      string
	Shard1DSN     string
	Shard2DSN     string
	RedisAddr     string
}

func Load() *Config {
	return &Config{
		ServerTimeout: getEnvAsDuration("SERVER_TIMEOUT", 15*time.Second),
		WorkerCount:   getEnvAsInt("WORKER_COUNT", runtime.NumCPU()),
		AppPort:       getEnv("APP_PORT", ":8080"),
		LogLevel:      getEnv("LOG_LEVEL", "debug"),
		Shard1DSN:     getEnv("SHARD_1_DSN", "postgres://postgres:postgres@localhost:5431/user_core_shard_1?sslmode=disable"),
		Shard2DSN:     getEnv("SHARD_2_DSN", "postgres://postgres:postgres@localhost:5432/user_core_shard_2?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsDuration(name string, fallback time.Duration) time.Duration {
	if valStr, exists := os.LookupEnv(name); exists {
		if duration, err := time.ParseDuration(valStr); err == nil {
			return duration
		}
	}
	return fallback
}

func getEnvAsInt(name string, fallback int) int {
	if valStr, exists := os.LookupEnv(name); exists {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return fallback
}
