package service

import (
	"strings"

	"github.com/benpate/derp"
)

// ParseStream parses a URL and returns the Stream token and action token.
func (service *Locator) ParseStream(url string) (string, string, error) {

	const location = "service.Locator.ParseStream"

	// Require that the URL starts with the host prefix
	if !strings.HasPrefix(url, service.host+"/") {
		return "", "", derp.NotFound(location, "URL must match host", "url: "+url, "host: "+service.host)
	}

	// Remove the host name from the URL
	url = strings.TrimPrefix(url, service.host+"/")

	// Isolate the stream token and action token
	streamToken, action, _ := strings.Cut(url, "/")

	// RULE: Stream token must not look like a username
	if strings.HasPrefix(url, "@") {
		return "", "", derp.NotFound(location, "Stream ID must not begin with '@'.")
	}

	// RULE: Stream token must not look like a "hidden" path
	if strings.HasPrefix(url, ".") {
		return "", "", derp.NotFound(location, "Stream ID must not begin with '.'", "url: "+url)
	}

	// RULE: Stream token must not be a reserved token
	if isReservedPath(streamToken) {
		return "", "", derp.NotFound(location, "Stream ID must not be a reserved token", "streamToken: "+streamToken)
	}

	// Remove any additional path segments from the action token
	action, _, _ = strings.Cut(action, "/")
	action, _, _ = strings.Cut(action, "?")

	// RULE: An empty stream token resolves to the "home" stream (mirrors legacy ParsePath).
	if streamToken == "" {
		streamToken = "home"
	}

	// RULE: An empty action resolves to the "view" action (mirrors legacy ParsePath).
	if action == "" {
		action = "view"
	}

	return streamToken, action, nil
}

// isReservedPath returns TRUE if the path value provided is reserved.
func isReservedPath(value string) bool {

	switch value {
	case "admin", "startup", "oauth", "signin", "signout", "register":
		return true
	}

	return false
}
