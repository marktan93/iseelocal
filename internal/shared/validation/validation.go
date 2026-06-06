package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"iseelocal/internal/shared/contracts"
)

type LocalTarget = contracts.LocalTarget

var subdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])$`)
var hostLabelPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var sensitivePorts = map[int]string{
	22:    "SSH",
	3306:  "MySQL",
	5432:  "PostgreSQL",
	6379:  "Redis",
	27017: "MongoDB",
}

func NormalizeSubdomain(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if len(value) < 2 || len(value) > 63 {
		return "", fmt.Errorf("subdomain must be between 2 and 63 characters")
	}
	if !subdomainPattern.MatchString(value) {
		return "", fmt.Errorf("subdomain must be a DNS label using letters, numbers, or hyphens")
	}
	return value, nil
}

func NormalizeUpstreamHost(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return "", nil
	}
	if strings.ContainsAny(value, "/\\:@ \t\r\n") {
		return "", fmt.Errorf("upstream host must be a hostname without scheme, port, path, or whitespace")
	}
	if len(value) > 253 {
		return "", fmt.Errorf("upstream host must be 253 characters or fewer")
	}
	labels := strings.Split(value, ".")
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 {
			return "", fmt.Errorf("upstream host must contain valid DNS labels")
		}
		if !hostLabelPattern.MatchString(label) {
			return "", fmt.Errorf("upstream host must contain valid DNS labels")
		}
	}
	return value, nil
}

func ValidateLocalTarget(target LocalTarget, allowSensitive bool) error {
	protocol := strings.ToLower(strings.TrimSpace(target.Protocol))
	if protocol == "" {
		protocol = "http"
	}
	if protocol != "http" {
		return fmt.Errorf("only http targets are supported in the MVP")
	}
	if target.Port < 1 || target.Port > 65535 {
		return fmt.Errorf("local port must be between 1 and 65535")
	}
	if service, ok := sensitivePorts[target.Port]; ok && !allowSensitive {
		return fmt.Errorf("local port %d (%s) is blocked by default", target.Port, service)
	}
	if !isAllowedLoopbackHost(target.Host) {
		return fmt.Errorf("local host must be loopback, got %q", target.Host)
	}
	return nil
}

func SensitivePortName(port int) (string, bool) {
	name, ok := sensitivePorts[port]
	return name, ok
}

func isAllowedLoopbackHost(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "localhost" {
		return true
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return false
	}
	if ip.String() == "169.254.169.254" {
		return false
	}
	return ip.IsLoopback()
}
