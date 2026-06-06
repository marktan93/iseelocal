package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	RelayAPIURL string          `json:"relay_api_url"`
	APIToken    string          `json:"api_token"`
	SSHHost     string          `json:"ssh_host"`
	SSHUser     string          `json:"ssh_user"`
	Routes      []TunnelMapping `json:"routes"`
}

type TunnelMapping struct {
	ID         string `json:"id"`
	Subdomain  string `json:"subdomain"`
	PublicURL  string `json:"public_url"`
	LocalHost  string `json:"local_host"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Protocol   string `json:"protocol"`
	Status     string `json:"status"`
}

func Default() Config {
	return Config{
		SSHUser: "tunnel",
		Routes:  []TunnelMapping{},
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.SSHUser == "" {
		cfg.SSHUser = "tunnel"
	}
	if cfg.Routes == nil {
		cfg.Routes = []TunnelMapping{}
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
