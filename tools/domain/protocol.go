package domain

import "strings"

// Protocol returns the appropriate protocol for a givin hostname.
// Local domains return `http://`, while all other domains return `https://`
func Protocol(hostname string) string {
	if IsLocalhost(hostname) {
		return "http://"
	}
	return "https://"
}

// NameOnly removes the protocol and port from a hostname
func NameOnly(host string) string {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	host = strings.Split(host, ":")[0]

	return host
}

func IsLocalhost(hostname string) bool {

	// Nornalize the hostname
	hostname = strings.ToLower(hostname)
	hostname = strings.TrimPrefix(hostname, "http://")
	hostname = strings.TrimPrefix(hostname, "https://")

	if hostname == "localhost" {
		return true
	}

	if hostname == "127.0.0.1" {
		return true
	}

	if strings.HasSuffix(hostname, ".local") {
		return true
	}

	if strings.HasPrefix(hostname, "10.") {
		return true
	}

	if strings.HasPrefix(hostname, "192.168") {
		return true
	}

	return false
}
