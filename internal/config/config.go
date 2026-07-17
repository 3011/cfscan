package config

import (
	"os"
	"strconv"
	"time"
)

type Server struct {
	HTTPAddr               string
	DatabaseURL            string
	AgentToken             string
	BootstrapAdminUser     string
	BootstrapAdminPassword string
	SessionTTL             time.Duration
	CookieSecure           bool
}

type Agent struct {
	CenterURL         string
	Token             string
	Name              string
	Region            string
	Continent         string
	Concurrency       int
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
}

func LoadServer() Server {
	return Server{
		HTTPAddr:               env("CFSCAN_HTTP_ADDR", ":8080"),
		DatabaseURL:            env("CFSCAN_DATABASE_URL", "postgres://cfscan:cfscan@localhost:5432/cfscan?sslmode=disable"),
		AgentToken:             env("CFSCAN_AGENT_TOKEN", "change-me"),
		BootstrapAdminUser:     env("CFSCAN_BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPassword: env("CFSCAN_BOOTSTRAP_ADMIN_PASSWORD", ""),
		SessionTTL:             envDuration("CFSCAN_SESSION_TTL", 24*time.Hour),
		CookieSecure:           envBool("CFSCAN_COOKIE_SECURE", true),
	}
}

func LoadAgent() Agent {
	return Agent{
		CenterURL:         env("CFSCAN_CENTER_URL", "http://localhost:8080"),
		Token:             env("CFSCAN_AGENT_TOKEN", "change-me"),
		Name:              env("CFSCAN_AGENT_NAME", "local-agent"),
		Region:            env("CFSCAN_AGENT_REGION", "local"),
		Continent:         env("CFSCAN_AGENT_CONTINENT", "local"),
		Concurrency:       envInt("CFSCAN_AGENT_CONCURRENCY", 64),
		HeartbeatInterval: envDuration("CFSCAN_AGENT_HEARTBEAT_INTERVAL", 15*time.Second),
		PollInterval:      envDuration("CFSCAN_AGENT_POLL_INTERVAL", 5*time.Second),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
