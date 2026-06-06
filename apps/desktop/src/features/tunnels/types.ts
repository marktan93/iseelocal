export type TunnelStatus = "offline" | "starting" | "online" | "stopping" | "error";

export interface TunnelRoute {
  id: string;
  subdomain: string;
  publicUrl: string;
  localHost: string;
  localPort: number;
  remoteHost: string;
  remotePort: number;
  protocol: "http";
  status: TunnelStatus;
}

export interface CreateRouteInput {
  subdomain: string;
  localHost: string;
  localPort: number;
  protocol: "http";
}

export interface TunnelClient {
  createRoute(input: CreateRouteInput): Promise<TunnelRoute>;
  startTunnel(route: TunnelRoute): Promise<void>;
  stopTunnel(route: TunnelRoute): Promise<void>;
  checkLocalTarget(host: string, port: number): Promise<boolean>;
}
