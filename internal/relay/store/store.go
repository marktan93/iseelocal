package store

import (
	"errors"

	"iseelocal/internal/shared/contracts"
)

var ErrNotFound = errors.New("route not found")

type Store interface {
	CreateRoute(route contracts.Route) error
	ListRoutes() ([]contracts.Route, error)
	ListUsedRemotePorts() (map[int]bool, error)
	GetRouteByHost(host string) (contracts.Route, error)
	GetRouteByID(id string) (contracts.Route, error)
	DeleteRoute(id string) error
	Heartbeat(id string) error
}
