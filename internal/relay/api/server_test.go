package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		SSHPort:    2222,
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

	if created.PublicURL != "https://myapp.example.com" || created.RemotePort != 18080 || created.SSHPort != 2222 {
		t.Fatalf("unexpected response: %#v", created)
	}
}

func TestServerTLSAskAllowsBaseDomainSubdomains(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{BaseDomain: "iseelocal.dev"}, st, ports.NewAllocator(18080, 18090))
	for _, domain := range []string{"iseelocal.dev", "api.iseelocal.dev", "bookkeeping-system.iseelocal.dev"} {
		req := httptest.NewRequest(http.MethodGet, "/api/tls-ask?domain="+domain, nil)
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		if res.Code != http.StatusNoContent {
			t.Fatalf("expected %s to be allowed, got %d", domain, res.Code)
		}
	}
}

func TestServerTLSAskRejectsOtherDomains(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{BaseDomain: "iseelocal.dev"}, st, ports.NewAllocator(18080, 18090))
	req := httptest.NewRequest(http.MethodGet, "/api/tls-ask?domain=attacker.example.com", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestServerRendersDashboard(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC()
	route := contracts.Route{
		ID:           "route_dashboard",
		Subdomain:    "phpmyadmin",
		PublicHost:   "phpmyadmin.152.42.204.9.sslip.io",
		PublicURL:    "http://phpmyadmin.152.42.204.9.sslip.io",
		ProjectName:  "phpMyAdmin",
		ProjectPath:  "/Users/whoami/Desktop/scripts/phpmyadmin",
		LocalHost:    "127.0.0.1",
		LocalPort:    80,
		UpstreamHost: "phpmyadmin.test",
		RemoteHost:   "127.0.0.1",
		RemotePort:   18080,
		Protocol:     "http",
		Status:       contracts.RouteStatusOnline,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := st.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	server := NewServer(Config{BaseDomain: "iseelocal.dev", PublicScheme: "https", SSHHost: "152.42.204.9", SSHPort: 2222}, st, ports.NewAllocator(18080, 18090))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "iseelocal VPS Dashboard") ||
		!strings.Contains(body, "phpMyAdmin") ||
		!strings.Contains(body, "iseelocal.dev") ||
		!strings.Contains(body, "152.42.204.9:2222") ||
		!strings.Contains(body, "/Users/whoami/Desktop/scripts/phpmyadmin") ||
		!strings.Contains(body, "Herd virtual host") ||
		!strings.Contains(body, "phpmyadmin.test") ||
		!strings.Contains(body, "https://phpmyadmin.iseelocal.dev") ||
		!strings.Contains(body, ".preview { display:block;") ||
		!strings.Contains(body, ".preview iframe { position:absolute;") {
		t.Fatalf("dashboard did not include expected route details: %s", body)
	}
	if strings.Contains(body, "152.42.204.9.sslip.io") {
		t.Fatalf("dashboard should derive public URLs from current base domain: %s", body)
	}
}

func TestServerCreatesRouteWithUpstreamHost(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{BaseDomain: "example.com", SSHHost: "vps.example.com", SSHUser: "tunnel"}, st, ports.NewAllocator(18080, 18090))

	body := bytes.NewBufferString(`{"subdomain":"phpmyadmin","project_name":"phpMyAdmin","project_path":"/Users/whoami/Desktop/scripts/phpmyadmin","local_host":"127.0.0.1","local_port":80,"upstream_host":"PhpMyAdmin.Test","protocol":"http","allow_sensitive_target":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routes", body)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}

	route, err := st.GetRouteByHost("phpmyadmin.example.com")
	if err != nil {
		t.Fatalf("GetRouteByHost returned error: %v", err)
	}
	if route.UpstreamHost != "phpmyadmin.test" {
		t.Fatalf("expected upstream host phpmyadmin.test, got %q", route.UpstreamHost)
	}
	if route.ProjectName != "phpMyAdmin" || route.ProjectPath != "/Users/whoami/Desktop/scripts/phpmyadmin" {
		t.Fatalf("unexpected project metadata: %#v", route)
	}
}

func TestServerCreatesRouteWithHTTPPublicScheme(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	server := NewServer(Config{BaseDomain: "example.com", PublicScheme: "http", SSHHost: "vps.example.com", SSHUser: "tunnel"}, st, ports.NewAllocator(18080, 18090))

	body := bytes.NewBufferString(`{"subdomain":"myapp","local_host":"127.0.0.1","local_port":3000,"protocol":"http"}`)
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
	if created.PublicURL != "http://myapp.example.com" {
		t.Fatalf("unexpected public URL: %q", created.PublicURL)
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
