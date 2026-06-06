import { invoke } from "@tauri-apps/api/core";

import type { CreateRouteInput, TunnelClient, TunnelRoute } from "../features/tunnels/types";

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}

let demoPort = 18080;

export function createDefaultTunnelClient(): TunnelClient {
  if (typeof window !== "undefined" && window.__TAURI_INTERNALS__) {
    return tauriClient;
  }
  return demoClient;
}

const tauriClient: TunnelClient = {
  async createRoute(input) {
    return invoke<TunnelRoute>("create_route", { input });
  },
  async startTunnel(route) {
    await invoke("start_tunnel", { route });
  },
  async stopTunnel(route) {
    await invoke("stop_tunnel", { routeId: route.id });
  },
  async checkLocalTarget(host, port) {
    return invoke<boolean>("check_local_target", { host, port });
  },
};

const demoClient: TunnelClient = {
  async createRoute(input: CreateRouteInput) {
    await delay(160);
    const subdomain = input.subdomain.trim().toLowerCase();
    return {
      id: `route_${crypto.randomUUID?.() ?? Date.now()}`,
      subdomain,
      publicUrl: `https://${subdomain}.example.com`,
      localHost: input.localHost,
      localPort: input.localPort,
      remoteHost: "127.0.0.1",
      remotePort: demoPort++,
      protocol: "http",
      status: "offline",
    };
  },
  async startTunnel() {
    await delay(220);
  },
  async stopTunnel() {
    await delay(160);
  },
  async checkLocalTarget() {
    await delay(80);
    return true;
  },
};

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
