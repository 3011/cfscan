package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type agentIdentity struct {
	ServerURL   string `json:"server_url"`
	AgentID     string `json:"agent_id"`
	Token       string `json:"token"`
	Name        string `json:"name"`
	Concurrency int    `json:"concurrency"`
}

func defaultIdentityPath() string {
	if value := os.Getenv("CFSCAN_AGENT_IDENTITY_FILE"); value != "" {
		return value
	}
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "cfscan-agent", "identity.json")
	}
	return filepath.Join(".", ".cfscan-agent", "identity.json")
}

func loadIdentity(path string) (agentIdentity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agentIdentity{}, err
	}
	var identity agentIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return agentIdentity{}, fmt.Errorf("decode identity: %w", err)
	}
	if identity.ServerURL == "" || identity.AgentID == "" || identity.Token == "" || identity.Concurrency < 1 {
		return agentIdentity{}, errors.New("identity file is incomplete")
	}
	return identity, nil
}

func saveIdentity(path string, identity agentIdentity) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create identity directory: %w", err)
	}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("encode identity: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".identity-*")
	if err != nil {
		return fmt.Errorf("create temporary identity: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect identity file: %w", err)
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return fmt.Errorf("write identity: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync identity: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close identity: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("install identity: %w", err)
	}
	if directory, err := os.Open(dir); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
