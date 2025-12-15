package cimd

import "github.com/benpate/rosetta/sliceof"

// Metadata defines all of the fields from the CIMD spec, defined on: https://client.dev/clients
type Metadata struct {
	ClientID               string         `json:"client_id"`                          // REQUIRED - Must match the URL serving this document
	RedirectURIs           sliceof.String `json:"redirect_uris"`                      // REQUIRED - Array of exact redirect URIs for your application
	ClientName             string         `json:"client_name"`                        // Recommended - Human readable name for your application
	LogoURI                string         `json:"logo_uri"`                           // Recommended - URL to your application's logo
	ClientURI              string         `json:"client_uri"`                         // Recommended - URL to your application's homepage
	TOSURI                 string         `json:"tos_uri,omitzero"`                   // (Optional) - Terms of Service
	PolicyURI              string         `json:"policy_uri,omitzero"`                // (Optional) - Privacy Policy
	GrantTypes             sliceof.String `json:"grant_types,omitzero"`               // (Optional) - Supported grant types
	ResponseTypes          sliceof.String `json:"response_types,omitzero"`            // (Optional) - Supported response types
	PostLogoutRedirectURIs sliceof.String `json:"post_logout_redirect_uris,omitzero"` // (Optional) - Post-logout redirects
}
