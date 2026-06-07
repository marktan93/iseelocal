import { FormEvent, useMemo, useState } from "react";
import {
  Activity,
  Copy,
  Globe2,
  Link2,
  Play,
  Plus,
  RefreshCw,
  Server,
  Square,
  Trash2,
} from "lucide-react";

import { createDefaultTunnelClient } from "../../lib/tunnelClient";
import type { TunnelClient, TunnelRoute, TunnelStatus } from "./types";

interface TunnelDashboardProps {
  client?: TunnelClient;
  initialRoutes?: TunnelRoute[];
}

interface FormState {
  subdomain: string;
  localHost: string;
  localPort: string;
}

const blockedPorts = new Set([22, 3306, 5432, 6379, 27017]);

export function TunnelDashboard({ client, initialRoutes = [] }: TunnelDashboardProps) {
  const tunnelClient = useMemo(() => client ?? createDefaultTunnelClient(), [client]);
  const [routes, setRoutes] = useState<TunnelRoute[]>(initialRoutes);
  const [form, setForm] = useState<FormState>({
    subdomain: "",
    localHost: "127.0.0.1",
    localPort: "3000",
  });
  const [error, setError] = useState("");
  const [isAdding, setIsAdding] = useState(false);
  const [logs, setLogs] = useState<string[]>([
    "Ready. Tunnel ports bind to VPS loopback and Caddy handles public HTTPS.",
  ]);

  const onlineCount = routes.filter((route) => route.status === "online").length;

  async function handleAddMapping(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    const validationError = validateForm(form);
    if (validationError) {
      setError(validationError);
      return;
    }

    const localPort = Number(form.localPort);
    setIsAdding(true);
    try {
      const reachable = await tunnelClient.checkLocalTarget(form.localHost, localPort);
      if (!reachable) {
        setError("Local target is not reachable.");
        return;
      }

      const route = await tunnelClient.createRoute({
        subdomain: form.subdomain,
        localHost: form.localHost,
        localPort,
        protocol: "http",
      });
      setRoutes((current) => [route, ...current]);
      setLogs((current) => [`Added ${route.publicUrl} -> ${route.localHost}:${route.localPort}`, ...current]);
      setForm((current) => ({ ...current, subdomain: "" }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not create mapping.");
    } finally {
      setIsAdding(false);
    }
  }

  async function startRoute(route: TunnelRoute) {
    setRouteStatus(route.id, "starting");
    setLogs((current) => [`Starting ${route.subdomain} on remote port ${route.remotePort}`, ...current]);
    try {
      await tunnelClient.startTunnel(route);
      setRouteStatus(route.id, "online");
      setLogs((current) => [`Online ${route.publicUrl}`, ...current]);
    } catch (err) {
      setRouteStatus(route.id, "error");
      setLogs((current) => [`Start failed for ${route.subdomain}: ${messageFromError(err)}`, ...current]);
    }
  }

  async function stopRoute(route: TunnelRoute) {
    setRouteStatus(route.id, "stopping");
    setLogs((current) => [`Stopping ${route.subdomain}`, ...current]);
    try {
      await tunnelClient.stopTunnel(route);
      setRouteStatus(route.id, "offline");
      setLogs((current) => [`Offline ${route.publicUrl}`, ...current]);
    } catch (err) {
      setRouteStatus(route.id, "error");
      setLogs((current) => [`Stop failed for ${route.subdomain}: ${messageFromError(err)}`, ...current]);
    }
  }

  function removeRoute(route: TunnelRoute) {
    setRoutes((current) => current.filter((item) => item.id !== route.id));
    setLogs((current) => [`Removed ${route.publicUrl}`, ...current]);
  }

  async function copyUrl(route: TunnelRoute) {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(route.publicUrl);
    }
    setLogs((current) => [`Copied ${route.publicUrl}`, ...current]);
  }

  function setRouteStatus(id: string, status: TunnelStatus) {
    setRoutes((current) => current.map((route) => (route.id === id ? { ...route, status } : route)));
  }

  return (
    <main className="shell">
      <aside className="sidebar" aria-label="Workspace">
        <div className="brand">
          <div className="brand-mark">
            <Link2 size={20} />
          </div>
          <div>
            <strong>iseelocal</strong>
            <span>Private tunnel control</span>
          </div>
        </div>

        <div className="sidebar-section">
          <span className="section-label">Relay</span>
          <div className="metric-row">
            <Server size={18} />
            <span>152.42.204.9</span>
          </div>
          <div className="metric-row">
            <Globe2 size={18} />
            <span>*.iseelocal.dev</span>
          </div>
        </div>

        <div className="sidebar-section">
          <span className="section-label">Session</span>
          <div className="stat-grid">
            <div>
              <strong>{routes.length}</strong>
              <span>Routes</span>
            </div>
            <div>
              <strong>{onlineCount}</strong>
              <span>Online</span>
            </div>
          </div>
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>Tunnels</h1>
            <p>HTTP routes through SSH reverse forwarding.</p>
          </div>
          <button className="icon-button" type="button" aria-label="Refresh status">
            <RefreshCw size={18} />
          </button>
        </header>

        <section className="mapping-panel" aria-label="Add mapping">
          <form className="mapping-form" onSubmit={handleAddMapping}>
            <label>
              <span>Public name</span>
              <input
                value={form.subdomain}
                onChange={(event) => setForm((current) => ({ ...current, subdomain: event.target.value }))}
                placeholder="myapp"
              />
            </label>
            <label>
              <span>Local host</span>
              <input
                value={form.localHost}
                onChange={(event) => setForm((current) => ({ ...current, localHost: event.target.value }))}
              />
            </label>
            <label>
              <span>Local port</span>
              <input
                inputMode="numeric"
                value={form.localPort}
                onChange={(event) => setForm((current) => ({ ...current, localPort: event.target.value }))}
              />
            </label>
            <button className="primary-button" type="submit" disabled={isAdding}>
              <Plus size={18} />
              <span>{isAdding ? "Adding" : "Add mapping"}</span>
            </button>
          </form>
          {error ? <div className="error-banner">{error}</div> : null}
        </section>

        <section className="route-table" aria-label="Routes">
          <div className="table-header">
            <span>Public URL</span>
            <span>Local target</span>
            <span>Remote</span>
            <span>Status</span>
            <span>Actions</span>
          </div>
          {routes.length === 0 ? (
            <div className="empty-state">
              <Activity size={20} />
              <span>No mappings yet</span>
            </div>
          ) : (
            routes.map((routeItem) => (
              <div className="route-row" key={routeItem.id}>
                <div className="url-cell">
                  <strong>{routeItem.subdomain}</strong>
                  <span>{routeItem.publicUrl}</span>
                </div>
                <span>{routeItem.localHost}:{routeItem.localPort}</span>
                <span>{routeItem.remoteHost}:{routeItem.remotePort}</span>
                <StatusBadge status={routeItem.status} />
                <div className="action-group">
                  {routeItem.status === "online" || routeItem.status === "starting" ? (
                    <button type="button" className="icon-button danger" aria-label={`Stop ${routeItem.subdomain}`} onClick={() => stopRoute(routeItem)}>
                      <Square size={16} />
                    </button>
                  ) : (
                    <button type="button" className="icon-button success" aria-label={`Start ${routeItem.subdomain}`} onClick={() => startRoute(routeItem)}>
                      <Play size={16} />
                    </button>
                  )}
                  <button type="button" className="icon-button" aria-label={`Copy ${routeItem.subdomain} URL`} onClick={() => copyUrl(routeItem)}>
                    <Copy size={16} />
                  </button>
                  <button type="button" className="icon-button" aria-label={`Remove ${routeItem.subdomain}`} onClick={() => removeRoute(routeItem)}>
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            ))
          )}
        </section>

        <section className="logs" aria-label="Logs">
          <div className="logs-header">
            <span>Logs</span>
            <span>{logs.length}</span>
          </div>
          <div className="log-list">
            {logs.map((line, index) => (
              <code key={`${line}-${index}`}>{line}</code>
            ))}
          </div>
        </section>
      </section>
    </main>
  );
}

function StatusBadge({ status }: { status: TunnelStatus }) {
  const label = status.charAt(0).toUpperCase() + status.slice(1);
  return (
    <span className={`status-badge ${status}`} aria-label={`Route status ${label}`}>
      {label}
    </span>
  );
}

function validateForm(form: FormState) {
  const subdomain = form.subdomain.trim();
  if (!/^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]$/.test(subdomain) || subdomain.length < 2) {
    return "Public name must be a valid DNS label.";
  }

  const localPort = Number(form.localPort);
  if (!Number.isInteger(localPort) || localPort < 1 || localPort > 65535) {
    return "Local port must be between 1 and 65535.";
  }
  if (blockedPorts.has(localPort)) {
    return `Local port ${localPort} is blocked by default.`;
  }
  if (!["127.0.0.1", "localhost", "::1"].includes(form.localHost.trim())) {
    return "Local host must be loopback.";
  }
  return "";
}

function messageFromError(err: unknown) {
  return err instanceof Error ? err.message : "unknown error";
}
