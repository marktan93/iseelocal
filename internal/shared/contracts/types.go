package contracts

import "time"

type RouteStatus string

const (
	RouteStatusOffline RouteStatus = "offline"
	RouteStatusOnline  RouteStatus = "online"
)

type LocalTarget struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type Route struct {
	ID              string      `json:"id"`
	Subdomain       string      `json:"subdomain"`
	PublicHost      string      `json:"public_host"`
	PublicURL       string      `json:"public_url"`
	LocalHost       string      `json:"local_host"`
	LocalPort       int         `json:"local_port"`
	RemoteHost      string      `json:"remote_host"`
	RemotePort      int         `json:"remote_port"`
	Protocol        string      `json:"protocol"`
	Status          RouteStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	LastHeartbeatAt *time.Time  `json:"last_heartbeat_at,omitempty"`
}

type CreateRouteRequest struct {
	Subdomain            string `json:"subdomain"`
	LocalHost            string `json:"local_host"`
	LocalPort            int    `json:"local_port"`
	Protocol             string `json:"protocol"`
	AllowSensitiveTarget bool   `json:"allow_sensitive_target,omitempty"`
}

type CreateRouteResponse struct {
	ID         string `json:"id"`
	PublicURL  string `json:"public_url"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
	SSHUser    string `json:"ssh_user"`
	SSHHost    string `json:"ssh_host"`
}

type RoutesResponse struct {
	Routes []Route `json:"routes"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
