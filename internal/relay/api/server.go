package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"iseelocal/internal/relay/ports"
	"iseelocal/internal/relay/store"
	"iseelocal/internal/shared/contracts"
	"iseelocal/internal/shared/validation"
)

type Config struct {
	BaseDomain string
	SSHHost    string
	SSHUser    string
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
	route := contracts.Route{
		ID:         newRouteID(),
		Subdomain:  subdomain,
		PublicHost: strings.ToLower(publicHost),
		PublicURL:  "https://" + strings.ToLower(publicHost),
		LocalHost:  strings.TrimSpace(req.LocalHost),
		LocalPort:  req.LocalPort,
		RemoteHost: "127.0.0.1",
		RemotePort: remotePort,
		Protocol:   "http",
		Status:     contracts.RouteStatusOffline,
		CreatedAt:  now,
		UpdatedAt:  now,
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
