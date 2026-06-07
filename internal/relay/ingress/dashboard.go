package ingress

import (
	"html/template"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"iseelocal/internal/shared/contracts"
)

type dashboardRoute struct {
	Title         string
	PublicURL     string
	LocalTarget   string
	RemoteTarget  string
	Status        string
	StatusClass   string
	LastHeartbeat string
}

type dashboardData struct {
	UpdatedAt    string
	RouteCount   int
	OnlineCount  int
	PublicScheme string
	BaseDomain   string
	SSHHost      string
	Routes       []dashboardRoute
}

func (p *Proxy) dashboard(w http.ResponseWriter) {
	routes, err := p.store.ListRoutes()
	if err != nil {
		http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(routes, func(i int, j int) bool {
		return routes[i].UpdatedAt.After(routes[j].UpdatedAt)
	})

	data := dashboardData{
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		RouteCount:   len(routes),
		PublicScheme: "https",
		BaseDomain:   p.config.BaseDomain,
		SSHHost:      p.config.SSHHost,
		Routes:       make([]dashboardRoute, 0, len(routes)),
	}
	for _, route := range routes {
		if route.Status == contracts.RouteStatusOnline {
			data.OnlineCount++
		}
		data.Routes = append(data.Routes, newDashboardRoute(route, data.PublicScheme, data.BaseDomain))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := dashboardTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
	}
}

func newDashboardRoute(route contracts.Route, publicScheme string, baseDomain string) dashboardRoute {
	lastHeartbeat := "never"
	if route.LastHeartbeatAt != nil {
		lastHeartbeat = route.LastHeartbeatAt.UTC().Format("2006-01-02 15:04:05 UTC")
	}

	status := string(route.Status)
	return dashboardRoute{
		Title:         route.Subdomain,
		PublicURL:     dashboardPublicURL(route, publicScheme, baseDomain),
		LocalTarget:   netHostPort(route.LocalHost, route.LocalPort),
		RemoteTarget:  netHostPort(route.RemoteHost, route.RemotePort),
		Status:        status,
		StatusClass:   status,
		LastHeartbeat: lastHeartbeat,
	}
}

func dashboardPublicURL(route contracts.Route, scheme string, baseDomain string) string {
	subdomain := strings.TrimSpace(route.Subdomain)
	domain := strings.TrimPrefix(strings.TrimSpace(baseDomain), ".")
	if subdomain == "" || domain == "" {
		return route.PublicURL
	}
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	return scheme + "://" + subdomain + "." + domain
}

func netHostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="15">
  <title>iseelocal VPS Dashboard</title>
  <style>
    :root { color-scheme: dark; --bg:#07111f; --panel:#0f1b2d; --muted:#8ea3bd; --line:#1f314a; --ok:#28d17c; --off:#f5b84b; --bad:#ff6b6b; --text:#eaf2ff; --accent:#7cc7ff; }
    * { box-sizing:border-box; }
    body { margin:0; font-family:ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background:var(--bg); color:var(--text); }
    main { max-width:1320px; margin:0 auto; padding:24px 20px 36px; }
    header { display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:22px; }
    h1 { margin:0 0 8px; font-size:28px; line-height:1.1; }
    .muted { color:var(--muted); }
    .cards { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:12px; margin-bottom:22px; }
    .card, .route-card { background:var(--panel); border:1px solid var(--line); border-radius:16px; box-shadow:0 16px 40px rgba(0,0,0,.14); }
    .card { min-width:0; padding:16px; }
    .card strong { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:24px; line-height:1.15; margin-bottom:4px; }
    .route-grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(360px,1fr)); gap:16px; align-items:start; }
    .route-card { overflow:hidden; }
    .preview { display:block; position:relative; height:220px; overflow:hidden; background:#050b14; border-bottom:1px solid var(--line); }
    .preview iframe { position:absolute; top:0; left:0; width:1440px; height:880px; border:0; transform:scale(.27); transform-origin:0 0; pointer-events:none; background:white; }
    .preview-overlay { position:absolute; inset:0; box-shadow:inset 0 -70px 60px rgba(7,17,31,.75); pointer-events:none; }
    .route-body { padding:16px; }
    .route-title { display:flex; align-items:flex-start; justify-content:space-between; gap:12px; margin-bottom:10px; }
    .route-title h2 { margin:0; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:18px; line-height:1.25; }
    .meta { display:grid; gap:8px; margin-top:12px; }
    .meta-row { display:grid; grid-template-columns:92px minmax(0,1fr); gap:10px; align-items:start; }
    .label { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.07em; }
    .value { min-width:0; overflow-wrap:anywhere; }
    a { color:var(--accent); text-decoration:none; }
    a:hover { text-decoration:underline; }
    code { color:#c9d7e8; background:#091326; border:1px solid var(--line); border-radius:8px; padding:2px 6px; white-space:nowrap; }
    .status { display:inline-flex; align-items:center; flex:0 0 auto; gap:6px; font-weight:700; }
    .dot { width:9px; height:9px; border-radius:999px; background:var(--off); }
    .online .dot { background:var(--ok); box-shadow:0 0 14px var(--ok); }
    .offline .dot { background:var(--bad); }
    .empty { padding:18px; background:var(--panel); border:1px solid var(--line); border-radius:16px; }
    @media (max-width: 900px) { .cards { grid-template-columns:repeat(2,minmax(0,1fr)); } .route-grid { grid-template-columns:1fr; } }
    @media (max-width: 640px) { main { padding:20px 14px; } header { display:block; } .cards { grid-template-columns:1fr; } .route-grid { grid-template-columns:minmax(0,1fr); } .preview { height:170px; } .preview iframe { transform:scale(.2); } }
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
  <section class="cards" aria-label="Relay summary">
    <div class="card"><strong>{{.RouteCount}}</strong><span class="muted">Routes</span></div>
    <div class="card"><strong>{{.OnlineCount}}</strong><span class="muted">Online</span></div>
    <div class="card"><strong>{{.BaseDomain}}</strong><span class="muted">Base domain</span></div>
    <div class="card"><strong>{{.SSHHost}}</strong><span class="muted">Tunnel SSH</span></div>
  </section>
  {{if .Routes}}
  <section class="route-grid" aria-label="Routes">
    {{range .Routes}}
    <article class="route-card">
      <a class="preview" href="{{.PublicURL}}" target="_blank" rel="noreferrer" aria-label="Open {{.Title}}">
        <iframe src="{{.PublicURL}}" loading="lazy" title="{{.Title}} preview"></iframe>
        <span class="preview-overlay"></span>
      </a>
      <div class="route-body">
        <div class="route-title">
          <h2>{{.Title}}</h2>
          <span class="status {{.StatusClass}}"><span class="dot"></span>{{.Status}}</span>
        </div>
        <div class="meta">
          <div class="meta-row"><span class="label">Public</span><span class="value"><a href="{{.PublicURL}}" target="_blank" rel="noreferrer">{{.PublicURL}}</a></span></div>
          <div class="meta-row"><span class="label">Target</span><span class="value"><code>{{.LocalTarget}}</code></span></div>
          <div class="meta-row"><span class="label">Remote</span><span class="value"><code>{{.RemoteTarget}}</code></span></div>
          <div class="meta-row"><span class="label">Heartbeat</span><span class="value">{{.LastHeartbeat}}</span></div>
        </div>
      </div>
    </article>
    {{end}}
  </section>
  {{else}}
  <div class="empty">No routes yet.</div>
  {{end}}
</main>
</body>
</html>`))
