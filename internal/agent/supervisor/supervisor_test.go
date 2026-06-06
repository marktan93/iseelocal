package supervisor

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	agentssh "iseelocal/internal/agent/ssh"
)

func TestSupervisorStartsAndStopsCommand(t *testing.T) {
	sup := New(func(ctx context.Context, _ []agentssh.TunnelSpec) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "ISEELOCAL_HELPER_PROCESS=1")
		return cmd, nil
	})

	err := sup.Start(context.Background(), []agentssh.TunnelSpec{
		{SSHUser: "tunnel", SSHHost: "vps.example.com", RemoteHost: "127.0.0.1", RemotePort: 18080, LocalHost: "127.0.0.1", LocalPort: 3000},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if sup.State() != StateRunning {
		t.Fatalf("expected running state, got %q", sup.State())
	}
	if err := sup.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if sup.State() != StateStopped {
		t.Fatalf("expected stopped state, got %q", sup.State())
	}
}

func TestSupervisorRejectsDuplicateStart(t *testing.T) {
	sup := New(func(ctx context.Context, _ []agentssh.TunnelSpec) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "ISEELOCAL_HELPER_PROCESS=1")
		return cmd, nil
	})
	defer func() {
		_ = sup.Stop()
	}()

	err := sup.Start(context.Background(), []agentssh.TunnelSpec{
		{SSHUser: "tunnel", SSHHost: "vps.example.com", RemoteHost: "127.0.0.1", RemotePort: 18080, LocalHost: "127.0.0.1", LocalPort: 3000},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	err = sup.Start(context.Background(), []agentssh.TunnelSpec{
		{SSHUser: "tunnel", SSHHost: "vps.example.com", RemoteHost: "127.0.0.1", RemotePort: 18081, LocalHost: "127.0.0.1", LocalPort: 8000},
	})
	if err == nil {
		t.Fatal("expected duplicate Start to fail")
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("ISEELOCAL_HELPER_PROCESS") != "1" {
		return
	}
	for {
		time.Sleep(time.Second)
	}
}
