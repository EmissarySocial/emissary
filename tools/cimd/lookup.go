package cimd

import (
	"encoding/json"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/uri"
)

// maxDocumentSize is the most of a client_id's answer that is read; a CIMD document is a few hundred bytes
const maxDocumentSize = 64 * 1024

// GetMetadata fetches the Client ID Metadata Document published at a client_id.  It returns FALSE with no
// error when the client answers but publishes no document, so the caller can describe it from its URL
func GetMetadata(host string, clientID string, allowPrivateIPs bool) (Metadata, bool, error) {

	const location = "cimd.GetMetadata"

	// RULE: The client_id must be a valid URL client identifier, and is fetched in canonical form
	clientID, err := ParseClientID(clientID)

	if err != nil {
		return Metadata{}, false, derp.Wrap(err, location, "Invalid client_id")
	}

	// RULE: If we're running on a production host, then do not allow local clients
	if !uri.IsLocalHostname(host) && uri.IsLocalHostname(clientID) {
		return Metadata{}, false, derp.BadRequest(location, "Invalid Client ID. Local clients can only be accessed on development instances", host, clientID)
	}

	// Request the document, asking for JSON.  The body is read as bytes, so an HTML page is not a decoding error
	body := make([]byte, 0)

	txn := remote.
		Get(clientID).
		UserAgent("Emissary-CIMD-Lookup/1.0").
		Accept("application/json").
		AllowPrivateIPs(allowPrivateIPs).
		MaxResponseSize(maxDocumentSize).
		Result(&body)

	sendError := txn.Send()
	statusCode := txn.ResponseStatusCode()

	// RULE: A client that cannot be reached cannot receive the redirect either.  The cause carries no response body
	if statusCode == 0 {
		return Metadata{}, false, derp.Wrap(sendError, location, "Unable to reach client_id", clientID, derp.WithCode(http.StatusBadGateway))
	}

	// RULE: A server error is refused rather than recorded, so the next attempt fetches again.  The error
	// names the client_id and status, never the response, which would carry part of the body
	if statusCode >= http.StatusInternalServerError {
		return Metadata{}, false, derp.BadGateway(location, "client_id answered with a server error", clientID, statusCode)
	}

	// RULE: Only a client that answers 2xx exists.  A 404 or a redirect that leads nowhere names nothing to authorize
	if (statusCode < http.StatusOK) || (statusCode >= http.StatusMultipleChoices) {
		return Metadata{}, false, derp.BadRequest(location, "client_id did not answer successfully", clientID, statusCode)
	}

	// RULE: An answer that cannot be read, such as a body over the size limit, is refused.  The cause may quote the body
	if sendError != nil {
		return Metadata{}, false, derp.BadRequest(location, "client_id sent an answer that cannot be read", clientID)
	}

	// Anything that is not JSON, such as an HTML page, publishes no metadata
	result := Metadata{}

	if err := json.Unmarshal(body, &result); err != nil {
		return Metadata{}, false, nil
	}

	// So does JSON that names no client, such as an ActivityPub actor
	if result.ClientID == "" {
		return Metadata{}, false, nil
	}

	// RULE: A document is served at the client_id itself.  One reached through a redirect is another address's document
	if wasRedirected(txn.Response(), clientID) {
		return Metadata{}, false, derp.BadRequest(location, "Client ID Metadata Document must not be served through a redirect", clientID)
	}

	// RULE: Prevent impersonation: a document that names a different client is refused
	if documentClientID, err := ParseClientID(result.ClientID); (err != nil) || (documentClientID != clientID) {
		return Metadata{}, false, derp.BadRequest(location, "Client ID Metadata Document names a different client_id", clientID)
	}

	// Woot.
	return result, true, nil
}

// wasRedirected returns TRUE if the answer came from an address other than the client_id that was requested
func wasRedirected(response *http.Response, clientID string) bool {

	// With no request on record there is nothing to compare, so the answer is treated as the client's own
	if (response == nil) || (response.Request == nil) || (response.Request.URL == nil) {
		return false
	}

	// An address that is not a valid client_id, such as an IP address, is certainly not this one
	answeredBy, err := ParseClientID(response.Request.URL.String())

	if err != nil {
		return true
	}

	return answeredBy != clientID
}
