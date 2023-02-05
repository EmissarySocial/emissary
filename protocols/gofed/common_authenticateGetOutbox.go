package gofed

import (
	"context"
	"net/http"
)

// AuthenticateGetOutbox determines whether the request is for a GET call to the Actor's Outbox. The out Context is
// used in further library calls, so your app's behavior can be modified depending on the authenticated context,
// such as whether to serve private messages.
//
// If an error is returned, it is passed back to the caller of GetOutbox. In this case, the implementation must not
// write a response to the http.ResponseWriteras is expected that the client will do so when handling the error.
// The authenticated is ignored.
//
// If no error is returned, but authentication or authorization fails, then then authenticated must be false and
// error nil. It is expected that the implementation handles writing to the http.ResponseWriter in this case.
//
// Finally, if the authentication and authorization succeeds, then authenticated must be true and error nil. The
// request will continue to be processed.
func (common Common) AuthenticateGetOutbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {

	// TODO: HIGH: Do we need to authenticate access to the Outbox?
	return c, true, nil
}
