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
		ID:           "route_test",
		Subdomain:    "myapp",
		PublicHost:   "myapp.example.com",
		PublicURL:    "https://myapp.example.com",
		LocalHost:    "127.0.0.1",
		LocalPort:    3000,
		UpstreamHost: "myapp.test",
		RemoteHost:   "127.0.0.1",
		RemotePort:   18080,
		Protocol:     "http",
		Status:       contracts.RouteStatusOffline,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := store.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error: %v", err)
	}

	got, err := store.GetRouteByHost("myapp.example.com")
	if err != nil {
		t.Fatalf("GetRouteByHost returned error: %v", err)
	}

	if got.ID != route.ID || got.RemotePort != 18080 || got.UpstreamHost != "myapp.test" {
		t.Fatalf("unexpected route: %#v", got)
	}
}

func TestSQLiteStoreMigratesUpstreamHostColumn(t *testing.T) {
	dbPath := t.TempDir() + "/routes.db"
	store, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if _, err := store.db.Exec(`ALTER TABLE routes DROP COLUMN upstream_host`); err != nil {
		t.Fatalf("DROP COLUMN upstream_host returned error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	store, err = OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite after migration returned error: %v", err)
	}
	defer store.Close()

	route := contracts.Route{
		ID:           "route_test",
		Subdomain:    "myapp",
		PublicHost:   "myapp.example.com",
		PublicURL:    "https://myapp.example.com",
		LocalHost:    "127.0.0.1",
		LocalPort:    3000,
		UpstreamHost: "myapp.test",
		RemoteHost:   "127.0.0.1",
		RemotePort:   18080,
		Protocol:     "http",
		Status:       contracts.RouteStatusOffline,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := store.CreateRoute(route); err != nil {
		t.Fatalf("CreateRoute returned error after migration: %v", err)
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
