package store

import (
	"testing"
	"time"

	"iseelocal/internal/shared/contracts"
)

func TestSQLiteStoreCreatesAndLooksUpRoutes(t *testing.T) {
	store, err := OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer store.Close()

	route := contracts.Route{
		ID:         "route_test",
		Subdomain:  "myapp",
		PublicHost: "myapp.example.com",
		PublicURL:  "https://myapp.example.com",
		LocalHost:  "127.0.0.1",
		LocalPort:  3000,
		RemoteHost: "127.0.0.1",
		RemotePort: 18080,
		Protocol:   "http",
		Status:     contracts.RouteStatusOffline,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := store.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	got, err := store.GetRouteByHost("myapp.example.com")
	if err != nil {
		t.Fatalf("GetRouteByHost returned error: %v", err)
	}

	if got.ID != route.ID || got.RemotePort != 18080 {
		t.Fatalf("unexpected route: %#v", got)
	}
}

func TestSQLiteStoreHeartbeatMarksRouteOnline(t *testing.T) {
	store, err := OpenSQLite(t.TempDir() + "/routes.db")
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer store.Close()

	route := contracts.Route{
		ID:         "route_test",
		Subdomain:  "myapp",
		PublicHost: "myapp.example.com",
		PublicURL:  "https://myapp.example.com",
		LocalHost:  "127.0.0.1",
		LocalPort:  3000,
		RemoteHost: "127.0.0.1",
		RemotePort: 18080,
		Protocol:   "http",
		Status:     contracts.RouteStatusOffline,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := store.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	if err := store.Heartbeat("route_test"); err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}

	got, err := store.GetRouteByHost("myapp.example.com")
	if err != nil {
		t.Fatalf("GetRouteByHost returned error: %v", err)
	}

	if got.Status != contracts.RouteStatusOnline {
		t.Fatalf("expected online status, got %q", got.Status)
	}
	if got.LastHeartbeatAt == nil {
		t.Fatal("expected LastHeartbeatAt to be set")
	}
}
