import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { TunnelDashboard } from "./TunnelDashboard";
import type { TunnelClient, TunnelRoute } from "./types";

function client(overrides: Partial<TunnelClient> = {}): TunnelClient {
  return {
    createRoute: vi.fn(async (input) => ({
      id: "route_1",
      subdomain: input.subdomain.toLowerCase(),
      publicUrl: `https://${input.subdomain.toLowerCase()}.iseelocal.dev`,
      localHost: input.localHost,
      localPort: input.localPort,
      remoteHost: "127.0.0.1",
      remotePort: 18080,
      protocol: "http" as const,
      status: "offline" as const,
    })),
    startTunnel: vi.fn(async () => undefined),
    stopTunnel: vi.fn(async () => undefined),
    checkLocalTarget: vi.fn(async () => true),
    ...overrides,
  };
}

function route(overrides: Partial<TunnelRoute> = {}): TunnelRoute {
  return {
    id: "route_1",
    subdomain: "myapp",
    publicUrl: "https://myapp.iseelocal.dev",
    localHost: "127.0.0.1",
    localPort: 3000,
    remoteHost: "127.0.0.1",
    remotePort: 18080,
    protocol: "http",
    status: "offline",
    ...overrides,
  };
}

describe("TunnelDashboard", () => {
  it("shows the configured relay endpoint", () => {
    render(<TunnelDashboard client={client()} />);

    expect(screen.getByText("152.42.204.9")).toBeInTheDocument();
    expect(screen.getByText("*.iseelocal.dev")).toBeInTheDocument();
  });

  it("adds a mapping and shows the public URL", async () => {
    const api = client();
    const user = userEvent.setup();

    render(<TunnelDashboard client={api} />);

    await user.type(screen.getByLabelText("Public name"), "MyApp");
    await user.clear(screen.getByLabelText("Local port"));
    await user.type(screen.getByLabelText("Local port"), "3000");
    await user.click(screen.getByRole("button", { name: "Add mapping" }));

    expect(await screen.findByText("https://myapp.iseelocal.dev")).toBeInTheDocument();
    expect(api.createRoute).toHaveBeenCalledWith({
      subdomain: "MyApp",
      localHost: "127.0.0.1",
      localPort: 3000,
      protocol: "http",
    });
  });

  it("blocks sensitive local ports before creating a route", async () => {
    const api = client();
    const user = userEvent.setup();

    render(<TunnelDashboard client={api} />);

    await user.type(screen.getByLabelText("Public name"), "db");
    await user.clear(screen.getByLabelText("Local port"));
    await user.type(screen.getByLabelText("Local port"), "5432");
    await user.click(screen.getByRole("button", { name: "Add mapping" }));

    expect(await screen.findByText("Local port 5432 is blocked by default.")).toBeInTheDocument();
    expect(api.createRoute).not.toHaveBeenCalled();
  });

  it("starts a tunnel and updates status", async () => {
    const api = client();
    const user = userEvent.setup();

    render(<TunnelDashboard client={api} initialRoutes={[route()]} />);

    await user.click(screen.getByRole("button", { name: "Start myapp" }));

    await waitFor(() => {
      expect(screen.getByLabelText("Route status Online")).toBeInTheDocument();
    });
    expect(api.startTunnel).toHaveBeenCalledWith(route());
  });
});
