package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"iseelocal/internal/agent/healthcheck"
	agentssh "iseelocal/internal/agent/ssh"
	"iseelocal/internal/agent/supervisor"
	"iseelocal/internal/shared/contracts"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	switch args[0] {
	case "check":
		return check(args[1:])
	case "ssh-args":
		return printSSHArgs(args[1:])
	case "run-ssh":
		return runSSH(args[1:])
	default:
		return usage()
	}
}

func check(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	host := fs.String("host", "127.0.0.1", "local host")
	port := fs.Int("port", 3000, "local port")
	timeout := fs.Duration("timeout", 2*time.Second, "request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	target := contracts.LocalTarget{Host: *host, Port: *port, Protocol: "http"}
	if err := healthcheck.CheckHTTPTarget(context.Background(), target, *timeout); err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}

func printSSHArgs(args []string) error {
	spec, err := parseTunnelSpec("ssh-args", args)
	if err != nil {
		return err
	}
	sshArgs, err := agentssh.BuildArgs([]agentssh.TunnelSpec{spec})
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(sshArgs, " "))
	return nil
}

func runSSH(args []string) error {
	spec, err := parseTunnelSpec("run-ssh", args)
	if err != nil {
		return err
	}

	sup := supervisor.New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sup.Start(ctx, []agentssh.TunnelSpec{spec}); err != nil {
		return err
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	return sup.Stop()
}

func parseTunnelSpec(name string, args []string) (agentssh.TunnelSpec, error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	sshUser := fs.String("ssh-user", "tunnel", "SSH tunnel user")
	sshHost := fs.String("ssh-host", "", "SSH host")
	remoteHost := fs.String("remote-host", "127.0.0.1", "remote bind host")
	remotePort := fs.Int("remote-port", 0, "remote bind port")
	localHost := fs.String("local-host", "127.0.0.1", "local target host")
	localPort := fs.Int("local-port", 0, "local target port")
	if err := fs.Parse(args); err != nil {
		return agentssh.TunnelSpec{}, err
	}
	return agentssh.TunnelSpec{
		SSHUser:    *sshUser,
		SSHHost:    *sshHost,
		RemoteHost: *remoteHost,
		RemotePort: *remotePort,
		LocalHost:  *localHost,
		LocalPort:  *localPort,
	}, nil
}

func usage() error {
	return fmt.Errorf(`usage:
	iseelocal-agent check --host 127.0.0.1 --port 3000
	iseelocal-agent ssh-args --ssh-host vps.example.com --remote-port 18080 --local-port 3000
	iseelocal-agent run-ssh --ssh-host vps.example.com --remote-port 18080 --local-port 3000`)
}
