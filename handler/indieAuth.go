package handler

import (
	"net/http"
)

// advertiseIndieAuth adds the IndieAuth discovery links to a response from a domain that has a primary user
func advertiseIndieAuth(header http.Header, host string, hasPrimaryUser bool) {

	// RULE: Only a domain with a primary user has anyone for IndieAuth to vouch for
	if !hasPrimaryUser {
		return
	}

	// Add, never Set, so a handler's own Link header survives
	header.Add("Link", indieAuthLinks(host))
}

// indieAuthLinks returns the Link header value that names the metadata document and both endpoints
func indieAuthLinks(host string) string {

	// Bridgy Fed reads only the two legacy rels, and the IndieAuth spec prefers the first
	return "<" + host + "/.well-known/oauth-authorization-server>; rel=\"indieauth-metadata\", " +
		"<" + host + "/oauth/authorize>; rel=\"authorization_endpoint\", " +
		"<" + host + "/oauth/token>; rel=\"token_endpoint\""
}
