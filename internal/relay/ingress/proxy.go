package ingress

import (
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"iseelocal/internal/relay/store"
	"iseelocal/internal/shared/contracts"
)

type Config struct {
	MaxBodyBytes int64
}

type Proxy struct {
	store  store.Store
	config Config
}

func NewProxy(store store.Store, config Config) http.Handler {
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 10 << 20
	}
	return &Proxy{store: store, config: config}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := normalizeHost(r.Host)
	route, err := p.store.GetRouteByHost(host)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, http.StatusText(status), status)
		return
	}
	if route.Status != contracts.RouteStatusOnline {
		http.Error(w, "route offline", http.StatusServiceUnavailable)
		return
	}

	target := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(route.RemoteHost, strconv.Itoa(route.RemotePort)),
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", host)
		req.Header.Set("X-Iseelocal-Route", route.ID)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "route unavailable", http.StatusBadGateway)
	}

	r.Body = http.MaxBytesReader(w, r.Body, p.config.MaxBodyBytes)
	proxy.ServeHTTP(w, r)
}

func normalizeHost(host string) string {
	value := strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(value); err == nil {
		return h
	}
	return value
}
