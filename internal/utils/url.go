package utils

import (
	"net"
	"net/url"
)

// IsValidURL performs basic safety and format validation to
// protect the URL shortener from invalid or malicious URLs
func IsValidURL(raw string) bool {
	// Parse URL and ensure it's syntactically valid
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	// Only allow http or https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// URL must contain a host component
	if u.Host == "" {
		return false
	}

	// Resolve host to IP addresses
	ips, err := net.LookupIP(u.Host)
	if err != nil {
		return false
	}

	// Block internal, loopback, or private IP ranges
	// Prevents SSRF(Server-Side Request Forgery) attacks
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() {
			return false
		}
	}

	// All checks passed → URL seems valid and safe
	return true
}
