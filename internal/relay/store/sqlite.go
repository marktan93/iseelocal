package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"iseelocal/internal/shared/contracts"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &SQLiteStore{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) init() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS routes (
	id TEXT PRIMARY KEY,
	subdomain TEXT NOT NULL UNIQUE,
	public_host TEXT NOT NULL UNIQUE,
	public_url TEXT NOT NULL,
	local_host TEXT NOT NULL,
	local_port INTEGER NOT NULL,
	remote_host TEXT NOT NULL,
	remote_port INTEGER NOT NULL UNIQUE,
	protocol TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	last_heartbeat_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_routes_public_host ON routes(public_host);
`)
	return err
}

func (s *SQLiteStore) CreateRoute(route contracts.Route) error {
	_, err := s.db.Exec(`
INSERT INTO routes (
	id, subdomain, public_host, public_url, local_host, local_port,
	remote_host, remote_port, protocol, status, created_at, updated_at, last_heartbeat_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, route.ID, route.Subdomain, strings.ToLower(route.PublicHost), route.PublicURL, route.LocalHost, route.LocalPort,
		route.RemoteHost, route.RemotePort, route.Protocol, route.Status, formatTime(route.CreatedAt), formatTime(route.UpdatedAt), formatOptionalTime(route.LastHeartbeatAt))
	return err
}

func (s *SQLiteStore) ListRoutes() ([]contracts.Route, error) {
	rows, err := s.db.Query(`
SELECT id, subdomain, public_host, public_url, local_host, local_port,
	remote_host, remote_port, protocol, status, created_at, updated_at, last_heartbeat_at
FROM routes
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []contracts.Route
	for rows.Next() {
		route, err := scanRoute(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (s *SQLiteStore) ListUsedRemotePorts() (map[int]bool, error) {
	rows, err := s.db.Query(`SELECT remote_port FROM routes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	used := map[int]bool{}
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, err
		}
		used[port] = true
	}
	return used, rows.Err()
}

func (s *SQLiteStore) GetRouteByHost(host string) (contracts.Route, error) {
	return s.getRoute(`public_host = ?`, strings.ToLower(strings.TrimSpace(host)))
}

func (s *SQLiteStore) GetRouteByID(id string) (contracts.Route, error) {
	return s.getRoute(`id = ?`, id)
}

func (s *SQLiteStore) DeleteRoute(id string) error {
	res, err := s.db.Exec(`DELETE FROM routes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) Heartbeat(id string) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(`UPDATE routes SET status = ?, updated_at = ?, last_heartbeat_at = ? WHERE id = ?`,
		contracts.RouteStatusOnline, formatTime(now), formatTime(now), id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) getRoute(where string, arg any) (contracts.Route, error) {
	row := s.db.QueryRow(`
SELECT id, subdomain, public_host, public_url, local_host, local_port,
	remote_host, remote_port, protocol, status, created_at, updated_at, last_heartbeat_at
FROM routes
WHERE `+where+`
LIMIT 1
`, arg)
	route, err := scanRoute(row)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.Route{}, ErrNotFound
	}
	return route, err
}

type routeScanner interface {
	Scan(dest ...any) error
}

func scanRoute(scanner routeScanner) (contracts.Route, error) {
	var route contracts.Route
	var status string
	var createdAt string
	var updatedAt string
	var lastHeartbeat sql.NullString
	err := scanner.Scan(
		&route.ID,
		&route.Subdomain,
		&route.PublicHost,
		&route.PublicURL,
		&route.LocalHost,
		&route.LocalPort,
		&route.RemoteHost,
		&route.RemotePort,
		&route.Protocol,
		&status,
		&createdAt,
		&updatedAt,
		&lastHeartbeat,
	)
	if err != nil {
		return contracts.Route{}, err
	}

	route.Status = contracts.RouteStatus(status)
	route.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return contracts.Route{}, err
	}
	route.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return contracts.Route{}, err
	}
	if lastHeartbeat.Valid {
		value, err := parseTime(lastHeartbeat.String)
		if err != nil {
			return contracts.Route{}, err
		}
		route.LastHeartbeatAt = &value
	}
	return route, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func formatOptionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
