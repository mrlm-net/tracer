package console

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// targetToAddr normalizes a target (URL or host:port) into host:port for tcp/udp.
func targetToAddr(targetURL string, tracerType string) (string, error) {
	addr := targetURL
	if strings.Contains(targetURL, "://") {
		u, err := url.Parse(targetURL)
		if err != nil {
			return "", fmt.Errorf("invalid target %q: %w", targetURL, err)
		}
		host := u.Hostname()
		port := u.Port()
		if port == "" {
			switch u.Scheme {
			case "http":
				port = "80"
			case "https":
				port = "443"
			default:
				return "", fmt.Errorf("no port in target %q and unknown scheme %q", targetURL, u.Scheme)
			}
		}
		addr = net.JoinHostPort(host, port)
	} else if !strings.Contains(targetURL, ":") {
		return "", fmt.Errorf("%s tracer target must be host:port or a URL with scheme", tracerType)
	}
	return addr, nil
}
