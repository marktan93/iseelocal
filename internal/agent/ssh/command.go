package ssh

import (
	"fmt"
	"os/exec"
	"strconv"
)

type TunnelSpec struct {
	SSHUser    string
	SSHHost    string
	SSHPort    int
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
		if normalizedSSHPort(spec.SSHPort) != normalizedSSHPort(first.SSHPort) {
			return nil, fmt.Errorf("all tunnel mappings must use the same SSH port")
		}
		args = append(args, "-R", fmt.Sprintf("%s:%d:%s:%d", spec.RemoteHost, spec.RemotePort, spec.LocalHost, spec.LocalPort))
	}

	if port := normalizedSSHPort(first.SSHPort); port != 22 {
		args = append(args, "-p", strconv.Itoa(port))
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
	if port := normalizedSSHPort(spec.SSHPort); port < 1 || port > 65535 {
		return fmt.Errorf("ssh port must be between 1 and 65535")
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

func normalizedSSHPort(port int) int {
	if port == 0 {
		return 22
	}
	return port
}
