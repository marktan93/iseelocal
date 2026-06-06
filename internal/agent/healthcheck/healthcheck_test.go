package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"iseelocal/internal/shared/contracts"
)

func TestCheckHTTPTargetAcceptsReachableLoopbackServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test URL: %v", err)
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	target := contracts.LocalTarget{Host: host, Port: port, Protocol: "http"}
	if err := CheckHTTPTarget(context.Background(), target, time.Second); err != nil {
		t.Fatalf("expected target to be reachable: %v", err)
	}
}

func TestCheckHTTPTargetRejectsUnavailablePort(t *testing.T) {
	target := contracts.LocalTarget{Host: "127.0.0.1", Port: 1, Protocol: "http"}
	if err := CheckHTTPTarget(context.Background(), target, 50*time.Millisecond); err == nil {
		t.Fatal("expected unavailable port to return error")
	}
}
