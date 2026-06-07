package ingress

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"iseelocal/internal/relay/store"
	"iseelocal/internal/shared/contracts"
)

func TestProxyRoutesByHostToRemoteLoopbackPort(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Local-App", "yes")
		_, _ = w.Write([]byte("hello from local"))
	}))
	defer local.Close()

	port, err := strconv.Atoi(local.URL[stringsLastIndex(local.URL, ":")+1:])
	if err != nil {
		t.Fatalf("parse local server port: %v", err)
	}

	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	route := contracts.Route{
		ID:         "route_test",
		Subdomain:  "myapp",
		PublicHost: "myapp.example.com",
		PublicURL:  "https://myapp.example.com",
		LocalHost:  "127.0.0.1",
		LocalPort:  3000,
		RemoteHost: "127.0.0.1",
		RemotePort: port,
		Protocol:   "http",
		Status:     contracts.RouteStatusOnline,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := st.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	proxy := NewProxy(st, Config{MaxBodyBytes: 1024 * 1024})
	req := httptest.NewRequest(http.MethodGet, "http://myapp.example.com/", nil)
	req.Host = "myapp.example.com"
	res := httptest.NewRecorder()

	proxy.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Body.String() != "hello from local" {
		t.Fatalf("unexpected body: %q", res.Body.String())
	}
}

func TestProxyServesDashboardForBaseDomain(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC()
	heartbeat := now.Add(time.Minute)
	route := contracts.Route{
		ID:              "route_test",
		Subdomain:       "myapp",
		PublicHost:      "myapp.152.42.204.9.sslip.io",
		PublicURL:       "http://myapp.152.42.204.9.sslip.io",
		LocalHost:       "127.0.0.1",
		LocalPort:       3000,
		RemoteHost:      "127.0.0.1",
		RemotePort:      18080,
		Protocol:        "http",
		Status:          contracts.RouteStatusOnline,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastHeartbeatAt: &heartbeat,
	}
	if err := st.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	proxy := NewProxy(st, Config{
		MaxBodyBytes: 1024 * 1024,
		BaseDomain:   "iseelocal.dev",
		SSHHost:      "152.42.204.9",
	})
	req := httptest.NewRequest(http.MethodGet, "https://iseelocal.dev/", nil)
	req.Host = "iseelocal.dev"
	res := httptest.NewRecorder()

	proxy.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"iseelocal VPS Dashboard",
		"iseelocal.dev",
		"152.42.204.9",
		"https://myapp.iseelocal.dev",
		".preview { display:block;",
		".preview iframe { position:absolute;",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("dashboard body missing %q:\n%s", expected, body)
		}
	}
	if strings.Contains(body, "152.42.204.9.sslip.io") {
		t.Fatalf("dashboard should derive public URLs from current base domain:\n%s", body)
	}
}

func TestProxyReturnsNotFoundForUnknownHost(t *testing.T) {
	st, err := store.OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer st.Close()

	proxy := NewProxy(st, Config{MaxBodyBytes: 1024 * 1024})
	req := httptest.NewRequest(http.MethodGet, "http://missing.example.com/", nil)
	req.Host = "missing.example.com"
	res := httptest.NewRecorder()

	proxy.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func stringsLastIndex(s string, sep string) int {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}
