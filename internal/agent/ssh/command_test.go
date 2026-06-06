package ssh

import (
	"reflect"
	"testing"
)

func TestBuildArgsCreatesSafeReverseForward(t *testing.T) {
	spec := TunnelSpec{
		SSHUser:    "tunnel",
		SSHHost:    "vps.example.com",
		RemoteHost: "127.0.0.1",
		RemotePort: 18080,
		LocalHost:  "127.0.0.1",
		LocalPort:  3000,
	}

	got, err := BuildArgs([]TunnelSpec{spec})
	if err != nil {
		t.Fatalf("BuildArgs returned error: %v", err)
	}

	want := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-R", "127.0.0.1:18080:127.0.0.1:3000",
		"tunnel@vps.example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args\nwant: %#v\n got: %#v", want, got)
	}
}

func TestBuildArgsSupportsMultipleMappingsOnOneHost(t *testing.T) {
	specs := []TunnelSpec{
		{SSHUser: "tunnel", SSHHost: "vps.example.com", RemoteHost: "127.0.0.1", RemotePort: 18080, LocalHost: "127.0.0.1", LocalPort: 3000},
		{SSHUser: "tunnel", SSHHost: "vps.example.com", RemoteHost: "127.0.0.1", RemotePort: 18081, LocalHost: "127.0.0.1", LocalPort: 8000},
	}

	got, err := BuildArgs(specs)
	if err != nil {
		t.Fatalf("BuildArgs returned error: %v", err)
	}

	if got[8] != "127.0.0.1:18080:127.0.0.1:3000" || got[10] != "127.0.0.1:18081:127.0.0.1:8000" {
		t.Fatalf("expected two -R mappings, got %#v", got)
	}
	if got[len(got)-1] != "tunnel@vps.example.com" {
		t.Fatalf("expected final destination, got %#v", got)
	}
}

func TestBuildArgsSupportsCustomSSHPort(t *testing.T) {
	spec := TunnelSpec{
		SSHUser:    "tunnel",
		SSHHost:    "vps.example.com",
		SSHPort:    2222,
		RemoteHost: "127.0.0.1",
		RemotePort: 18080,
		LocalHost:  "127.0.0.1",
		LocalPort:  3000,
	}

	got, err := BuildArgs([]TunnelSpec{spec})
	if err != nil {
		t.Fatalf("BuildArgs returned error: %v", err)
	}

	want := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-R", "127.0.0.1:18080:127.0.0.1:3000",
		"-p", "2222",
		"tunnel@vps.example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args\nwant: %#v\n got: %#v", want, got)
	}
}

func TestBuildArgsRejectsMixedDestinations(t *testing.T) {
	specs := []TunnelSpec{
		{SSHUser: "tunnel", SSHHost: "one.example.com", RemoteHost: "127.0.0.1", RemotePort: 18080, LocalHost: "127.0.0.1", LocalPort: 3000},
		{SSHUser: "tunnel", SSHHost: "two.example.com", RemoteHost: "127.0.0.1", RemotePort: 18081, LocalHost: "127.0.0.1", LocalPort: 8000},
	}

	if _, err := BuildArgs(specs); err == nil {
		t.Fatal("expected mixed SSH hosts to be rejected")
	}
}
