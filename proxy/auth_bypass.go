package proxy

import (
	"net"
	"strings"

	"proxygate/config"
)

func canBypassProxyAuth(cfg *config.Config, remoteAddr string) bool {
	if cfg == nil || !cfg.LocalAuthBypass {
		return false
	}
	return isLoopbackRemoteAddr(remoteAddr)
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host := strings.TrimSpace(remoteAddr)
	if host == "" {
		return false
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
