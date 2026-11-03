package activitypub_user

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// accountDeletePeekLimit is how much of an inbox body is kept for recognizing an account deletion,
// which is always small
const accountDeletePeekLimit = 64 * 1024

// peekRequestBody returns up to limit bytes from the start of the request body, and puts them back
// so the request still reads the whole, unchanged body.
func peekRequestBody(request *http.Request, limit int64) []byte {

	if request.Body == nil {
		return nil
	}

	peeked, _ := io.ReadAll(io.LimitReader(request.Body, limit))
	request.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(peeked), request.Body), request.Body}

	return peeked
}

// isAccountSelfDelete reports whether body is an actor's Delete of itself -- the notice a server
// sends to every inbox it knows when one of its accounts is deleted.
func isAccountSelfDelete(body []byte) bool {

	var activity struct {
		Type   string          `json:"type"`
		Actor  string          `json:"actor"`
		Object json.RawMessage `json:"object"`
	}

	if err := json.Unmarshal(body, &activity); err != nil || activity.Type != "Delete" || activity.Actor == "" {
		return false
	}

	// The object is either the actor's id, or a document carrying it
	var objectID string

	if err := json.Unmarshal(activity.Object, &objectID); err != nil {

		var object struct {
			ID string `json:"id"`
		}

		if err := json.Unmarshal(activity.Object, &object); err != nil {
			return false
		}

		objectID = object.ID
	}

	return objectID == activity.Actor
}
