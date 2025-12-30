package netutil

import (
	"context"
	"net"
	"strings"
	"time"
)

// ParseAddr accepts inputs like "host", "host:port", "127.0.0.1", "[::1]:8080", "fe80::1%en0"
// and returns a normalized host, port, joinAddr (suitable for Dial), parsed IP (if literal),
// whether it was an IP literal, the zone (for IPv6 link-local) and an error.
func ParseAddr(input, defaultPort string) (host, port, joinAddr string, ip net.IP, isIP bool, zone string, err error) {
	host = input
	port = defaultPort

	// try split host:port (handles [::1]:8080)
	if h, p, e := net.SplitHostPort(input); e == nil {
		host = h
		port = p
	}

	// if host is bracketed IPv6 like [::1], remove brackets for parsing
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}

	// detect zone for IPv6 literals (fe80::1%en0)
	zone = ""
	if i := strings.LastIndex(host, "%"); i != -1 {
		zone = host[i+1:]
		host = host[:i]
	}

	// attempt parse IP (without zone)
	ip = net.ParseIP(host)
	if ip != nil {
		isIP = true
	}

	// joinAddr should include original host (with zone if present) to preserve link-local
	joinHost := host
	if zone != "" {
		joinHost = joinHost + "%" + zone
	}
	// For IPv6 literals without brackets, net.JoinHostPort will add brackets
	joinAddr = net.JoinHostPort(joinHost, port)

	return host, port, joinAddr, ip, isIP, zone, nil
}

// IsIPv4 returns true if ip (may be nil) is IPv4.
func IsIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6 returns true if ip is IPv6.
func IsIPv6(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}

// ResolveAndDial resolves host (if hostname) and attempts to dial in family-preferred order.
// networkBase is "tcp" or "udp". prefer can be "v4", "v6" or ""/"auto".
// Returns established connection, chosen IP, list of resolved IPs, chosen family ("v4"/"v6"), or error.
func ResolveAndDial(ctx context.Context, networkBase, host, port, prefer string, timeout time.Duration) (net.Conn, net.IP, []net.IP, string, error) {
	// If host is an IP literal, dial directly with appropriate family.
	if ip := net.ParseIP(host); ip != nil {
		var network string
		family := "v4"
		if IsIPv4(ip) {
			network = networkBase + "4"
			family = "v4"
		} else {
			network = networkBase + "6"
			family = "v6"
		}
		d := &net.Dialer{Timeout: timeout}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(host, port))
		return conn, ip, nil, family, err
	}

	// Otherwise resolve via DNS
	var resolved []net.IP
	// Use the default resolver
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, nil, nil, "", err
	}
	resolved = append(resolved, ips...)

	// Partition addresses
	var v4s, v6s []net.IP
	for _, ip := range resolved {
		if IsIPv4(ip) {
			v4s = append(v4s, ip)
		} else {
			v6s = append(v6s, ip)
		}
	}

	order := make([]net.IP, 0, len(resolved))
	pref := strings.ToLower(prefer)
	if pref == "v6" {
		order = append(order, v6s...)
		order = append(order, v4s...)
	} else if pref == "v4" {
		order = append(order, v4s...)
		order = append(order, v6s...)
	} else {
		// default: use returned order (platform resolver ordering)
		order = append(order, resolved...)
	}

	// Dial attempts: keep per-attempt timeout small to avoid long serial waits.
	perAttempt := 5 * time.Second
	if timeout > 0 && timeout < perAttempt {
		perAttempt = timeout
	}

	for _, ip := range order {
		var network string
		family := "v4"
		if IsIPv4(ip) {
			network = networkBase + "4"
			family = "v4"
		} else {
			network = networkBase + "6"
			family = "v6"
		}
		addr := net.JoinHostPort(ip.String(), port)
		d := &net.Dialer{Timeout: perAttempt}
		cctx, cancel := context.WithTimeout(ctx, perAttempt)
		conn, derr := d.DialContext(cctx, network, addr)
		cancel()
		if derr == nil {
			return conn, ip, resolved, family, nil
		}
		// otherwise try next
	}

	return nil, nil, resolved, "", context.DeadlineExceeded
}
