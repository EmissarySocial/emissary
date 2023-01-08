package gofed

import (
	"context"
	"net/http"

	"github.com/go-fed/activity/streams/vocab"
)

// GetOutbox returns a proper paginated view of the Outbox for serving in a response.
// Since AuthenticateGetOutbox is called before this, the implementation is responsible
// for ensuring things like proper pagination, visible content based on permissions,
// and whether to leverage the pub.Database's GetOutbox method in this implementation.
func (common *Common) GetOutbox(c context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error) {
	return nil, nil
}
