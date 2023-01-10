package gofed

import (
	"context"
	"net/http"

	"github.com/go-fed/activity/pub"
)

/*
This is a hook that occurs after reading the request body of a POST request to an Actor's Inbox.

Provides your application the opportunity to set contextual information based on the incoming http.Request and its body. Some applications simply return c and do nothing else, which is OK. More commonly, software simply inspects the http.Request path to determine the actual local Actor being interacted with, and save such information within c.

Any errors returned immediately abort processing of the request and are returned to the caller of the Actor's PostInbox.
*/
func (fed Federating) PostInboxRequestBodyHook(c context.Context, r *http.Request, activity pub.Activity) (context.Context, error) {
	// TODO: CRITICAL: Do This
	return c, nil
}
