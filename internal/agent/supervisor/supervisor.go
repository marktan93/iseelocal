package supervisor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	agentssh "iseelocal/internal/agent/ssh"
)

type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
)

type CommandFactory func(context.Context, []agentssh.TunnelSpec) (*exec.Cmd, error)

type Supervisor struct {
	mu      sync.Mutex
	factory CommandFactory
	state   State
	cancel  context.CancelFunc
	done    chan error
	logs    []string
}

func New(factory CommandFactory) *Supervisor {
	if factory == nil {
		factory = defaultCommandFactory
	}
	return &Supervisor{factory: factory, state: StateStopped, logs: []string{}}
}

func (s *Supervisor) Start(ctx context.Context, specs []agentssh.TunnelSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRunning {
		return fmt.Errorf("tunnel process is already running")
	}

	processCtx, cancel := context.WithCancel(ctx)
	cmd, err := s.factory(processCtx, specs)
	if err != nil {
		cancel()
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	s.cancel = cancel
	s.done = make(chan error, 1)
	s.state = StateRunning
	s.appendLogLocked("ssh tunnel process started")

	go s.collectLogs(stdout)
	go s.collectLogs(stderr)
	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		if s.state == StateRunning {
			s.state = StateStopped
		}
		if err != nil {
			s.appendLogLocked("ssh tunnel process exited: " + err.Error())
		} else {
			s.appendLogLocked("ssh tunnel process stopped")
		}
		s.mu.Unlock()
		s.done <- err
	}()

	return nil
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()
	if s.state != StateRunning {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for tunnel process to stop")
	}

	s.mu.Lock()
	s.state = StateStopped
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Supervisor) Logs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *Supervisor) collectLogs(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		s.mu.Lock()
		s.appendLogLocked(scanner.Text())
		s.mu.Unlock()
	}
}

func (s *Supervisor) appendLogLocked(line string) {
	s.logs = append(s.logs, time.Now().UTC().Format(time.RFC3339)+" "+line)
	if len(s.logs) > 500 {
		s.logs = s.logs[len(s.logs)-500:]
	}
}

func defaultCommandFactory(ctx context.Context, specs []agentssh.TunnelSpec) (*exec.Cmd, error) {
	args, err := agentssh.BuildArgs(specs)
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, "ssh", args...), nil
}
