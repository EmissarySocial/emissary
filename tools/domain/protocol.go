package domain

import "strings"

func Protocol(hostname string) string {

	if IsLocalhost(hostname) {
		return "http://"
	}
	return "https://"
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
