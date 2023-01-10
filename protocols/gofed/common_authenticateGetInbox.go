package gofed

import (
	"context"
	"net/http"
)

// AuthenticateGetInbox determines whether the request is for a GET call to the Actor's Inbox. The out
// Context is used in further library calls, so your app's behavior can be modified depending on the
// authenticated context, such as whether to serve private messages.
//
// If an error is returned, it is passed back to the caller of GetInbox. In this case, the implementation
// must not write a response to the http.ResponseWriter as is expected that the client will do so when
// handling the error. The authenticated is ignored.
//
// If no error is returned, but authentication or authorization fails, then authenticated must be false
// and error nil. It is expected that the implementation handles writing to the http.ResponseWriter in
// this case.
//
// Finally, if the authentication and authorization succeeds, then then authenticated must be true and
// error nil. The request will continue to be processed.
func (common Common) AuthenticateGetInbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {

	// TODO: CRITICAL: Do we need to allow access to the Inbox??
	w.Write([]byte("Access Denied."))
	return c, false, nil
}
