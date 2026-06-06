package ingress

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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
