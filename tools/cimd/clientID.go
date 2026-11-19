package cimd

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// maxClientIDLength bounds a client_id before it is parsed or fetched
const maxClientIDLength = 2048

// ParseClientID checks a URL client_id against the IndieAuth rules for a client identifier (§3.3),
// and returns it in canonical form: a lower-case scheme and host, and a path of at least "/"
func ParseClientID(value string) (string, error) {

	const location = "cimd.ParseClientID"

	// RULE: Bound the value before parsing it
	if len(value) > maxClientIDLength {
		return "", derp.BadRequest(location, "client_id is too long")
	}

	// RULE: No fragment, including an empty one that url.Parse would drop silently
	if strings.Contains(value, "#") {
		return "", derp.BadRequest(location, "client_id must not contain a fragment", value)
	}

	parsed, err := url.Parse(value)

	if err != nil {
		return "", derp.BadRequest(location, "client_id must be a URL", value)
	}

	// RULE: An http or https scheme, written as scheme://host
	parsed.Scheme = strings.ToLower(parsed.Scheme)

	if (parsed.Scheme != "http") && (parsed.Scheme != "https") {
		return "", derp.BadRequest(location, "client_id must use http or https", value)
	}

	if parsed.Opaque != "" {
		return "", derp.BadRequest(location, "client_id must be a URL with a host", value)
	}

	// RULE: No username or password
	if parsed.User != nil {
		return "", derp.BadRequest(location, "client_id must not contain a username or password", value)
	}

	// RULE: The host is a domain name, never an IP address
	hostname := strings.ToLower(parsed.Hostname())

	if uri.IsValidIPAddress(hostname) {
		return "", derp.BadRequest(location, "client_id must name a domain, not an IP address", value)
	}

	if err := uri.ValidateHostname(hostname); err != nil {
		return "", derp.BadRequest(location, "client_id must name a valid domain", value)
	}

	// RULE: No "." or ".." path segments
	for segment := range strings.SplitSeq(parsed.Path, "/") {
		if (segment == ".") || (segment == "..") {
			return "", derp.BadRequest(location, "client_id must not contain '.' or '..' path segments", value)
		}
	}

	// A URL with no path is treated as having the path "/" (IndieAuth §3.4)
	if parsed.Path == "" {
		parsed.Path = "/"
		parsed.RawPath = ""
	}

	parsed.Host = strings.ToLower(parsed.Host)
	parsed.ForceQuery = false

	// RULE: The canonical form is bounded too, because escaping can triple the length of a path
	result := parsed.String()

	if len(result) > maxClientIDLength {
		return "", derp.BadRequest(location, "client_id is too long")
	}

	// Canonical, and clean as a whistle
	return result, nil
}
