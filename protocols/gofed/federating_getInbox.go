package gofed

import (
	"context"
	"net/http"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// GetInbox returns a proper paginated view of the Inbox for serving in a response.
// Since AuthenticateGetInbox is called before this, the implementation is responsible
// for ensuring things like proper pagination, visible content based on permissions,
// and whether to leverage the pub.Database's GetInbox method in this implementation.
func (fed Federating) GetInbox(c context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error) {
	// Assuming we may not need GetInbox..
	// TODO: LOW: re-evaluate this assumption after the app is running.
	return streams.NewActivityStreamsOrderedCollectionPage(), nil
}
