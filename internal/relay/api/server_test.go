package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iseelocal/internal/relay/ports"
	"iseelocal/internal/relay/store"
	"iseelocal/internal/shared/contracts"
)

func TestServerCreatesRoute(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{
		BaseDomain: "example.com",
		SSHHost:    "vps.example.com",
		SSHUser:    "tunnel",
	}, st, ports.NewAllocator(18080, 18090))

	body := bytes.NewBufferString(`{"subdomain":"MyApp","local_host":"127.0.0.1","local_port":3000,"protocol":"http"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routes", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}

	var created contracts.CreateRouteResponse
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if created.PublicURL != "https://myapp.example.com" || created.RemotePort != 18080 {
		t.Fatalf("unexpected response: %#v", created)
	}
}

func TestServerRejectsSensitiveLocalPort(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{BaseDomain: "example.com", SSHHost: "vps.example.com", SSHUser: "tunnel"}, st, ports.NewAllocator(18080, 18090))

	body := strings.NewReader(`{"subdomain":"db","local_host":"127.0.0.1","local_port":5432,"protocol":"http"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routes", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
