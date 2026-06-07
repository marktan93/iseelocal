package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"iseelocal/internal/relay/api"
	"iseelocal/internal/relay/auth"
	"iseelocal/internal/relay/ingress"
	"iseelocal/internal/relay/ports"
	"iseelocal/internal/relay/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	st, err := store.OpenSQLite(cfg.databasePath)
	if err != nil {
		return fmt.Errorf("open sqlite store: %w", err)
	}
	defer st.Close()

	apiHandler := auth.BearerMiddleware(cfg.apiToken, api.NewServer(api.Config{
		BaseDomain: cfg.baseDomain,
		SSHHost:    cfg.sshHost,
		SSHUser:    cfg.sshUser,
	}, st, ports.NewAllocator(cfg.remotePortStart, cfg.remotePortEnd)))

	ingressHandler := accessLog(ingress.NewProxy(st, ingress.Config{
		MaxBodyBytes: 10 << 20,
		BaseDomain:   cfg.baseDomain,
		SSHHost:      cfg.sshHost,
	}))

	apiServer := &http.Server{Addr: cfg.apiAddr, Handler: apiHandler}
	ingressServer := &http.Server{Addr: cfg.ingressAddr, Handler: ingressHandler}

	errs := make(chan error, 2)
	go func() {
		log.Printf("relay API listening on %s", cfg.apiAddr)
		errs <- apiServer.ListenAndServe()
	}()
	go func() {
		log.Printf("relay ingress listening on %s", cfg.ingressAddr)
		errs <- ingressServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Printf("received %s, shutting down", sig)
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = apiServer.Shutdown(ctx)
	_ = ingressServer.Shutdown(ctx)
	return nil
}

type config struct {
	apiToken        string
	baseDomain      string
	sshHost         string
	sshUser         string
	databasePath    string
	apiAddr         string
	ingressAddr     string
	remotePortStart int
	remotePortEnd   int
}

func loadConfig() (config, error) {
	cfg := config{
		apiToken:        os.Getenv("ISEELOCAL_API_TOKEN"),
		baseDomain:      os.Getenv("ISEELOCAL_BASE_DOMAIN"),
		sshHost:         os.Getenv("ISEELOCAL_SSH_HOST"),
		sshUser:         getenv("ISEELOCAL_SSH_USER", "tunnel"),
		databasePath:    getenv("ISEELOCAL_DATABASE", "./iseelocal.db"),
		apiAddr:         getenv("ISEELOCAL_API_ADDR", "127.0.0.1:8081"),
		ingressAddr:     getenv("ISEELOCAL_INGRESS_ADDR", "127.0.0.1:8080"),
		remotePortStart: getenvInt("ISEELOCAL_REMOTE_PORT_START", 18080),
		remotePortEnd:   getenvInt("ISEELOCAL_REMOTE_PORT_END", 18999),
	}
	if cfg.apiToken == "" {
		return config{}, fmt.Errorf("ISEELOCAL_API_TOKEN is required")
	}
	if cfg.baseDomain == "" {
		return config{}, fmt.Errorf("ISEELOCAL_BASE_DOMAIN is required")
	}
	if cfg.sshHost == "" {
		return config{}, fmt.Errorf("ISEELOCAL_SSH_HOST is required")
	}
	return cfg, nil
}

func getenv(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func getenvInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("ingress host=%s method=%s path=%s status=%d duration=%s", r.Host, r.Method, r.URL.Path, rec.status, time.Since(started))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
