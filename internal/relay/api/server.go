package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"iseelocal/internal/relay/ports"
	"iseelocal/internal/relay/store"
	"iseelocal/internal/shared/contracts"
	"iseelocal/internal/shared/validation"
)

type Config struct {
	BaseDomain   string
	PublicScheme string
	SSHHost      string
	SSHUser      string
	SSHPort      int
}

type Server struct {
	config    Config
	store     store.Store
	allocator ports.Allocator
}

func NewServer(config Config, store store.Store, allocator ports.Allocator) http.Handler {
	return &Server{config: config, store: store, allocator: allocator}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case (r.URL.Path == "/" || r.URL.Path == "/dashboard") && r.Method == http.MethodGet:
		s.dashboard(w, r)
	case r.URL.Path == "/api/health" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, contracts.HealthResponse{Status: "ok"})
	case r.URL.Path == "/api/routes" && r.Method == http.MethodPost:
		s.createRoute(w, r)
	case r.URL.Path == "/api/routes" && r.Method == http.MethodGet:
		s.listRoutes(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/routes/"):
		s.routeAction(w, r)
	default:
		writeJSON(w, http.StatusNotFound, contracts.ErrorResponse{Error: "not found"})
	}
}

func (s *Server) dashboard(w http.ResponseWriter, _ *http.Request) {
	routes, err := s.store.ListRoutes()
	if err != nil {
		http.Error(w, "failed to list routes", http.StatusInternalServerError)
		return
	}

	data := dashboardData{
		BaseDomain: s.config.BaseDomain,
		SSHHost:    s.config.SSHHost,
		SSHPort:    s.config.SSHPort,
		Routes:     routes,
		Online:     countRoutes(routes, contracts.RouteStatusOnline),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
	}
}

type dashboardData struct {
	BaseDomain string
	SSHHost    string
	SSHPort    int
	Routes     []contracts.Route
	Online     int
	UpdatedAt  string
}

func countRoutes(routes []contracts.Route, status contracts.RouteStatus) int {
	count := 0
	for _, route := range routes {
		if route.Status == status {
			count++
		}
	}
	return count
}

var dashboardTemplate = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"targetHost": func(route contracts.Route) string {
		if route.UpstreamHost != "" {
			return route.UpstreamHost
		}
		return fmt.Sprintf("%s:%d", route.LocalHost, route.LocalPort)
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="15">
  <title>iseelocal VPS Dashboard</title>
  <style>
    :root { color-scheme: dark; --bg:#07111f; --panel:#0f1b2d; --muted:#8ea3bd; --line:#1f314a; --ok:#28d17c; --off:#f5b84b; --text:#eaf2ff; }
    body { margin:0; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background:var(--bg); color:var(--text); }
    main { max-width:1180px; margin:0 auto; padding:32px 20px; }
    header { display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:24px; }
    h1 { margin:0 0 8px; font-size:28px; }
    .muted { color:var(--muted); }
    .cards { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:12px; margin-bottom:22px; }
    .card, table { background:var(--panel); border:1px solid var(--line); border-radius:16px; }
    .card { padding:16px; }
    .card strong { display:block; font-size:24px; margin-bottom:4px; }
    table { width:100%; border-collapse:separate; border-spacing:0; overflow:hidden; }
    th, td { padding:13px 14px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
    th { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.08em; }
    tr:last-child td { border-bottom:0; }
    a { color:#7cc7ff; text-decoration:none; }
    a:hover { text-decoration:underline; }
    code { color:#c9d7e8; background:#091326; border:1px solid var(--line); border-radius:8px; padding:2px 6px; white-space:nowrap; }
    .status { display:inline-flex; align-items:center; gap:6px; font-weight:700; }
    .dot { width:9px; height:9px; border-radius:999px; background:var(--off); }
    .online .dot { background:var(--ok); box-shadow:0 0 14px var(--ok); }
    @media (max-width: 900px) { .cards { grid-template-columns:repeat(2,minmax(0,1fr)); } table { font-size:14px; } }
    @media (max-width: 640px) { header { display:block; } .cards { grid-template-columns:1fr; } th:nth-child(3), td:nth-child(3), th:nth-child(4), td:nth-child(4) { display:none; } }
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>iseelocal VPS Dashboard</h1>
      <div class="muted">Readonly route and tunnel status. Auto-refreshes every 15 seconds.</div>
    </div>
    <div class="muted">Updated {{.UpdatedAt}}</div>
  </header>
  <section class="cards">
    <div class="card"><strong>{{len .Routes}}</strong><span class="muted">Routes</span></div>
    <div class="card"><strong>{{.Online}}</strong><span class="muted">Online</span></div>
    <div class="card"><strong>{{.BaseDomain}}</strong><span class="muted">Base domain</span></div>
    <div class="card"><strong>{{.SSHHost}}:{{.SSHPort}}</strong><span class="muted">Tunnel SSH</span></div>
  </section>
  <table>
    <thead>
      <tr><th>Status</th><th>Public URL</th><th>Local target</th><th>VPS remote</th><th>Last heartbeat</th></tr>
    </thead>
    <tbody>
      {{range .Routes}}
      <tr>
        <td><span class="status {{.Status}}"><span class="dot"></span>{{.Status}}</span></td>
        <td><a href="{{.PublicURL}}" target="_blank" rel="noreferrer">{{.PublicURL}}</a><br><span class="muted">{{.Subdomain}}</span></td>
        <td><code>{{targetHost .}}</code><br><span class="muted">{{.LocalHost}}:{{.LocalPort}}</span></td>
        <td><code>{{.RemoteHost}}:{{.RemotePort}}</code></td>
        <td>{{if .LastHeartbeatAt}}{{.LastHeartbeatAt.Format "2006-01-02 15:04:05 UTC"}}{{else}}<span class="muted">never</span>{{end}}</td>
      </tr>
      {{else}}
      <tr><td colspan="5" class="muted">No routes yet.</td></tr>
      {{end}}
    </tbody>
  </table>
</main>
</body>
</html>`))

func (s *Server) createRoute(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req contracts.CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.ErrorResponse{Error: "invalid JSON body"})
		return
	}

	subdomain, err := validation.NormalizeSubdomain(req.Subdomain)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.ErrorResponse{Error: err.Error()})
		return
	}

	target := validation.LocalTarget{Host: req.LocalHost, Port: req.LocalPort, Protocol: req.Protocol}
	if target.Protocol == "" {
		target.Protocol = "http"
	}
	if err := validation.ValidateLocalTarget(target, req.AllowSensitiveTarget); err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.ErrorResponse{Error: err.Error()})
		return
	}
	upstreamHost, err := validation.NormalizeUpstreamHost(req.UpstreamHost)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, contracts.ErrorResponse{Error: err.Error()})
		return
	}

	used, err := s.store.ListUsedRemotePorts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, contracts.ErrorResponse{Error: "failed to read used ports"})
		return
	}
	remotePort, err := s.allocator.Next(used)
	if err != nil {
		writeJSON(w, http.StatusConflict, contracts.ErrorResponse{Error: err.Error()})
		return
	}

	now := time.Now().UTC()
	publicHost := fmt.Sprintf("%s.%s", subdomain, strings.TrimPrefix(strings.TrimSpace(s.config.BaseDomain), "."))
	publicScheme := s.config.PublicScheme
	if publicScheme == "" {
		publicScheme = "https"
	}
	route := contracts.Route{
		ID:           newRouteID(),
		Subdomain:    subdomain,
		PublicHost:   strings.ToLower(publicHost),
		PublicURL:    publicScheme + "://" + strings.ToLower(publicHost),
		LocalHost:    strings.TrimSpace(req.LocalHost),
		LocalPort:    req.LocalPort,
		UpstreamHost: upstreamHost,
		RemoteHost:   "127.0.0.1",
		RemotePort:   remotePort,
		Protocol:     "http",
		Status:       contracts.RouteStatusOffline,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.CreateRoute(route); err != nil {
		writeJSON(w, http.StatusConflict, contracts.ErrorResponse{Error: "route already exists or remote port is unavailable"})
		return
	}

	writeJSON(w, http.StatusCreated, contracts.CreateRouteResponse{
		ID:         route.ID,
		PublicURL:  route.PublicURL,
		RemoteHost: route.RemoteHost,
		RemotePort: route.RemotePort,
		SSHUser:    s.config.SSHUser,
		SSHHost:    s.config.SSHHost,
		SSHPort:    s.config.SSHPort,
	})
}

func (s *Server) listRoutes(w http.ResponseWriter, _ *http.Request) {
	routes, err := s.store.ListRoutes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, contracts.ErrorResponse{Error: "failed to list routes"})
		return
	}
	writeJSON(w, http.StatusOK, contracts.RoutesResponse{Routes: routes})
}

func (s *Server) routeAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/routes/")
	if rest == "" {
		writeJSON(w, http.StatusNotFound, contracts.ErrorResponse{Error: "not found"})
		return
	}

	if strings.HasSuffix(rest, "/heartbeat") && r.Method == http.MethodPost {
		id := strings.TrimSuffix(rest, "/heartbeat")
		if err := s.store.Heartbeat(id); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, contracts.ErrorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == http.MethodDelete {
		if err := s.store.DeleteRoute(rest); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, contracts.ErrorResponse{Error: err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusNotFound, contracts.ErrorResponse{Error: "not found"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func newRouteID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("route_%d", time.Now().UnixNano())
	}
	return "route_" + hex.EncodeToString(bytes[:])
}
