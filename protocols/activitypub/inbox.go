package activitypub

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

func ParseInboxRequest(request *http.Request) (mapof.Any, error) {

	result := mapof.NewAny()

	// RULE: Content-Type MUST be "application/activity+json"
	if request.Header.Get(ContentType) != ContentTypeActivityPub {
		return result, derp.New(400, "activitypub.HandleInboxPost", "Content-Type MUST be 'application/activity+json'")
	}

	// TODO: Verify the request signature
	// RULE: Verify request signatures
	// verifier, err := httpsig.NewVerifier(request)

	// Try to read the body from the request
	bodyReader, err := request.GetBody()

	if err != nil {
		return result, derp.Wrap(err, "activitypub.HandleInboxPost", "Error copying request body")
	}

	// Try to read the body into the buffer
	var bodyBuffer bytes.Buffer

	if _, err = bodyBuffer.ReadFrom(bodyReader); err != nil {
		return result, derp.Wrap(err, "activitypub.HandleInboxPost", "Error reading body into buffer")
	}

	// Try to unmarshal the body from the buffer into a map.

	if err := json.Unmarshal(bodyBuffer.Bytes(), &result); err != nil {
		return result, derp.Wrap(err, "activitypub.HandleInboxPost", "Error unmarshalling body")
	}

	// Return the activity to the caller.
	return result, nil
}
