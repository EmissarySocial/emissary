package cimd

import (
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/remote"
)

// GetMetadata retrieves the CIMD metadata from the provided URL
func GetMetadata(host string, url string, options ...remote.Option) (Metadata, error) {

	const location = "cimd.GetMetadata"

	// RULE: If we're running on a production host, then do not allow local clients
	if !dt.IsLocalhost(host) && dt.IsLocalhost(url) {
		return Metadata{}, derp.BadRequest(location, "Invalid Client ID. Local clients can only be accessed on development instances", host, url)
	}

	// Stage the HTTP request
	result := Metadata{}
	txn := remote.
		Get(url).
		UserAgent("Emissary-CIMD-Lookup/1.0").
		With(options...).
		Result(&result)

	// Send the HTTP request
	if err := txn.Send(); err != nil {
		return Metadata{}, derp.Wrap(err, location, "Unable to retrieve CIMD metadata", url)
	}

	// RULE: Prevent impersonation attacks by validating the returned Client ID against the requested URL
	if result.ClientID != url {
		return Metadata{}, derp.BadRequest(location, "Client ID must match the provided URL", url, result.ClientID)
	}

	// Woot.
	return result, nil
}
