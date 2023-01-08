package gofed

import (
	"context"
	"net/url"
)

/*
Given a list of actorIRIs, determines whether any are blocked for this particular request context and based on the particular application's state. For example, some applications allow users or software instances to maintain lists of blocked peer Actors or domains.

To determine the current user being interacted with, it is recommended to set such information in the PostInboxRequestBodyHook method.

If an error is returned, it is passed back to the caller of PostInbox.

If no error is returned, but the interaction should be blocked, then blocked must be true and err nil. An http.StatusForbidden will be written in the response.

Finally, the interaction should proceed, then blocked must be false and err nil. The request will continue to be processed.
*/
func (fed Federating) Blocked(c context.Context, actorIRIs []*url.URL) (blocked bool, err error) {
	return false, nil
}
