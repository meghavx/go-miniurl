package utils

import (
	"net"
	"net/http"
	"strings"
)

// GetIP returns the real client IP, even when behind reverse proxies
func GetIP(r *http.Request) string {
	// 1. Check X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}

	// 2. Check X-Real-IP (some proxies use this)
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.Split(xrip, ",")[0]
	}

	// 3. Fallback: extract IP from r.RemoteAddr (format: "IP:port")
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	// 4. Final fallback (rare): return entire RemoteAddr
	return r.RemoteAddr
}
