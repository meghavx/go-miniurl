package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var chars = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func Base62Encode(num uint64) string {
	num = 100_000_000 - num

	res := make([]byte, 0)
	for num > 0 {
		res = append(res, chars[num%62])
		num /= 62
	}
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

func Base62Decode(s string) uint64 {
	var num uint64
	for i := 0; i < len(s); i++ {
		num = num*62 + uint64(bytes.IndexByte(chars, s[i]))
	}
	return 100_000_000 - num
}

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

// ValidateLongURL checks whether a given URL is safe and valid
func ValidateLongURL(rawURL string) (string, error) {
	if !isSafeURL(rawURL) {
		return "", errors.New("Invalid or unsafe URL")
	}
	return rawURL, nil
}

// ValidateShortURL checks whether a given URL is
// recognized by this URL shortener service or not
func ValidateShortURL(rawURL string, serverHost string) (string, error) {
	// Parse URL and ensure it's syntactically valid
	u, ok := isValidURLFormat(rawURL)
	if !ok {
		return "", errors.New("Invalid URL format")
	}

	// Host MUST match server's host (key check)
	if u.Host != serverHost {
		return "", errors.New("Invalid/Unrecognized short URL")
	}

	// Extract code "/abc123"
	code := u.Path[1:]
	if code == "" {
		return "", errors.New("Incomplete short URL: No short code found")
	}
	return code, nil
}

// HashURL returns an SHA-256 hash of the URL as a hex string
// Safe for use in Redis keys and database storage
func HashURL(rawURL string) string {
	hash := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(hash[:])
}

/**** Helper Methods below ****/

// isSafeURL performs basic safety and format validation to
// protect the URL shortener from invalid or malicious URLs
func isSafeURL(rawURL string) bool {
	u, ok := isValidURLFormat(rawURL)
	if !ok {
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

// isValidURLFormat checks basic syntax + scheme + host
func isValidURLFormat(rawURL string) (*url.URL, bool) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, false
	}

	// Only allow http or https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false
	}

	// URL must contain a host component
	if u.Host == "" {
		return nil, false
	}

	return u, true
}