package ssh

import (
	"fmt"
	"os/exec"
)

type TunnelSpec struct {
	SSHUser    string
	SSHHost    string
	RemoteHost string
	RemotePort int
	LocalHost  string
	LocalPort  int
}

func BuildArgs(specs []TunnelSpec) ([]string, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one tunnel mapping is required")
	}

	first := specs[0]
	if err := validateSpec(first); err != nil {
		return nil, err
	}

	args := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
	}

	for _, spec := range specs {
		if err := validateSpec(spec); err != nil {
			return nil, err
		}
		if spec.SSHUser != first.SSHUser || spec.SSHHost != first.SSHHost {
			return nil, fmt.Errorf("all tunnel mappings must use the same SSH destination")
		}
		args = append(args, "-R", fmt.Sprintf("%s:%d:%s:%d", spec.RemoteHost, spec.RemotePort, spec.LocalHost, spec.LocalPort))
	}

	args = append(args, fmt.Sprintf("%s@%s", first.SSHUser, first.SSHHost))
	return args, nil
}

func Command(specs []TunnelSpec) (*exec.Cmd, error) {
	args, err := BuildArgs(specs)
	if err != nil {
		return nil, err
	}
	return exec.Command("ssh", args...), nil
}

func validateSpec(spec TunnelSpec) error {
	if spec.SSHUser == "" {
		return fmt.Errorf("ssh user is required")
	}
	if spec.SSHHost == "" {
		return fmt.Errorf("ssh host is required")
	}
	if spec.RemoteHost == "" {
		return fmt.Errorf("remote host is required")
	}
	if spec.LocalHost == "" {
		return fmt.Errorf("local host is required")
	}
	if spec.RemotePort < 1 || spec.RemotePort > 65535 {
		return fmt.Errorf("remote port must be between 1 and 65535")
	}
	if spec.LocalPort < 1 || spec.LocalPort > 65535 {
		return fmt.Errorf("local port must be between 1 and 65535")
	}
	return nil
}
