// Package httputil contains small HTTP-layer helpers that operate on
// *http.Request values, which the pure-string github.com/benpate/uri
// package deliberately does not depend on.
package httputil

import "net/http"

// TrueHostname returns the host name from the request, accounting for
// proxy headers (like X-Forwarded-Host).
func TrueHostname(request *http.Request) string {

	// If this is a proxied request, then use the X-Forwarded-Host header
	// instead of the Host header
	if trueHost := request.Header.Get("X-Forwarded-Host"); trueHost != "" {
		return trueHost
	}

	// Fallback to the Host header if X-Forwarded-Host is not present
	return request.Host
}
