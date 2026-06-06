use std::net::{TcpStream, ToSocketAddrs};
use std::sync::atomic::{AtomicU16, Ordering};
use std::time::Duration;

use serde::{Deserialize, Serialize};

static NEXT_REMOTE_PORT: AtomicU16 = AtomicU16::new(18080);

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct CreateRouteInput {
    subdomain: String,
    local_host: String,
    local_port: u16,
    protocol: String,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
#[serde(rename_all = "camelCase")]
struct TunnelRoute {
    id: String,
    subdomain: String,
    public_url: String,
    local_host: String,
    local_port: u16,
    remote_host: String,
    remote_port: u16,
    protocol: String,
    status: String,
}

#[tauri::command]
fn create_route(input: CreateRouteInput) -> Result<TunnelRoute, String> {
    let subdomain = normalize_subdomain(&input.subdomain)?;
    if input.protocol != "http" {
        return Err("only http routes are supported in the MVP".to_string());
    }
    if is_sensitive_port(input.local_port) {
        return Err(format!(
            "local port {} is blocked by default",
            input.local_port
        ));
    }

    let remote_port = NEXT_REMOTE_PORT.fetch_add(1, Ordering::SeqCst);
    Ok(TunnelRoute {
        id: format!("route_{}", remote_port),
        subdomain: subdomain.clone(),
        public_url: format!("https://{}.example.com", subdomain),
        local_host: input.local_host,
        local_port: input.local_port,
        remote_host: "127.0.0.1".to_string(),
        remote_port,
        protocol: "http".to_string(),
        status: "offline".to_string(),
    })
}

#[tauri::command]
fn check_local_target(host: String, port: u16) -> bool {
    let addr = format!("{}:{}", host, port);
    let Ok(mut addrs) = addr.to_socket_addrs() else {
        return false;
    };
    let Some(addr) = addrs.next() else {
        return false;
    };
    TcpStream::connect_timeout(&addr, Duration::from_millis(900)).is_ok()
}

#[tauri::command]
fn start_tunnel(route: TunnelRoute) -> Result<(), String> {
    validate_route(&route)
}

#[tauri::command]
fn stop_tunnel(route_id: String) -> Result<(), String> {
    if route_id.trim().is_empty() {
        return Err("route id is required".to_string());
    }
    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            create_route,
            check_local_target,
            start_tunnel,
            stop_tunnel
        ])
        .run(tauri::generate_context!())
        .expect("error while running iseelocal");
}

fn normalize_subdomain(input: &str) -> Result<String, String> {
    let value = input.trim().to_lowercase();
    if value.len() < 2 || value.len() > 63 {
        return Err("public name must be between 2 and 63 characters".to_string());
    }
    let valid = value
        .chars()
        .all(|ch| ch.is_ascii_lowercase() || ch.is_ascii_digit() || ch == '-')
        && !value.starts_with('-')
        && !value.ends_with('-');
    if !valid {
        return Err("public name must be a valid DNS label".to_string());
    }
    Ok(value)
}

fn is_sensitive_port(port: u16) -> bool {
    matches!(port, 22 | 3306 | 5432 | 6379 | 27017)
}

fn validate_route(route: &TunnelRoute) -> Result<(), String> {
    if route.remote_host != "127.0.0.1" {
        return Err("remote host must bind to 127.0.0.1".to_string());
    }
    if route.local_host != "127.0.0.1" && route.local_host != "localhost" && route.local_host != "::1" {
        return Err("local host must be loopback".to_string());
    }
    if is_sensitive_port(route.local_port) {
        return Err(format!(
            "local port {} is blocked by default",
            route.local_port
        ));
    }
    Ok(())
}
