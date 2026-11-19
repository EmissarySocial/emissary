package mastodon

import (
	"net/http"

	"github.com/benpate/derp"
)

// errNotSupported is the 422 for creating or changing something Emissary does not have, such as a filter
func errNotSupported(location string, what string) error {

	// Apps show this message, where a 501 "not implemented" reads as a crash
	return derp.BadRequest(location, what+" are not supported on this server yet", derp.WithCode(http.StatusUnprocessableEntity))
}

// errNoSuch is the 404 for reading or removing one of something Emissary never creates, such as a poll
func errNoSuch(location string, what string) error {
	return derp.NotFound(location, what+" not found")
}
