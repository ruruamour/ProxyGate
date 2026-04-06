package proxy

import (
	"net/netip"
	"net/url"
	"strings"
)

var nonPublicDomainSuffixes = []string{
	".localhost",
	".local",
	".home.arpa",
	".internal",
	".lan",
	".test",
	".invalid",
	".example",
}

func shouldPenalizeProxyForTarget(target string) bool {
	host := normalizeTargetHost(target)
	if host == "" {
		return false
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" || host == "localhost" {
		return false
	}
	for _, suffix := range nonPublicDomainSuffixes {
		if strings.HasSuffix(host, suffix) {
			return false
		}
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return isPublicTargetAddr(addr)
	}

	return strings.Contains(host, ".")
}

func normalizeTargetHost(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	if strings.Contains(target, "://") {
		u, err := url.Parse(target)
		if err != nil {
			return ""
		}
		return u.Hostname()
	}

	if strings.HasPrefix(target, "/") {
		return ""
	}

	if u, err := url.Parse("http://" + target); err == nil {
		return u.Hostname()
	}

	return ""
}

func isPublicTargetAddr(addr netip.Addr) bool {
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return false
	}

	if !addr.Is4() {
		return true
	}

	v4 := addr.As4()
	switch {
	case v4[0] == 0:
		return false
	case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127:
		return false
	case v4[0] == 192 && v4[1] == 0 && v4[2] == 0:
		return false
	case v4[0] == 192 && v4[1] == 0 && v4[2] == 2:
		return false
	case v4[0] == 198 && (v4[1] == 18 || v4[1] == 19):
		return false
	case v4[0] == 198 && v4[1] == 51 && v4[2] == 100:
		return false
	case v4[0] == 203 && v4[1] == 0 && v4[2] == 113:
		return false
	}

	return true
}
