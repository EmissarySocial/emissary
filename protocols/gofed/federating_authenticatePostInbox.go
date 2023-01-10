package gofed

import (
	"context"
	"net/http"
)

// AuthenticatePostInbox is a callback for your application to determine whether the incoming
// http.Request is authenticated and, implicitly, authorized to proceed with processing the
// request.
//
// If an error is returned, it is passed back to the caller of PostInbox. In this case, the
// implementation must not write a response to the http.ResponseWriter as is expected that
// the client will do so when handling the error. The authenticated value is ignored in this
// case.
//
// If no error is returned, but your application determines that authentication or
// authorization fails, then authenticated must be false and err nil. It is expected that the
// implementation handles writing to the http.ResponseWriter in this case.
//
// Finally, if the authentication and authorization succeeds, then authenticated must be true
// and err nil. The request will continue to be processed.
func (fed Federating) AuthenticatePostInbox(c context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	// TODO: HIGH: Implement Blocks here.
	return nil, false, nil
}
