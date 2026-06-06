package config

import "testing"

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	path := t.TempDir() + "/config.json"
	cfg := Config{
		RelayAPIURL: "https://api.example.com",
		APIToken:    "secret",
		SSHHost:     "vps.example.com",
		SSHUser:     "tunnel",
		Routes: []TunnelMapping{
			{ID: "route_1", Subdomain: "myapp", PublicURL: "https://myapp.example.com", LocalHost: "127.0.0.1", LocalPort: 3000, RemotePort: 18080, Protocol: "http", Status: "offline"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.RelayAPIURL != cfg.RelayAPIURL || len(got.Routes) != 1 || got.Routes[0].RemotePort != 18080 {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadMissingConfigReturnsDefault(t *testing.T) {
	got, err := Load(t.TempDir() + "/missing.json")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.SSHUser != "tunnel" {
		t.Fatalf("expected default SSH user tunnel, got %q", got.SSHUser)
	}
}
