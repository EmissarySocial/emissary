package content

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
)

// Adapter reads content from one kind of remote source
type Adapter interface {

	// Protocol returns the StreamSource Method that this Adapter reads
	Protocol() string

	// Version returns an opaque token for the current state of the remote source.  It is cheap:
	// one round trip, and no content.
	Version(ctx context.Context, source model.StreamSource) (string, error)

	// Fetch returns the content that the remote source holds at the provided Version
	Fetch(ctx context.Context, source model.StreamSource, version string) (Item, error)

	// Subscribe asks the remote source to push change notifications to a callback URL, or returns
	// a NotImplemented error when the source cannot push
	Subscribe(ctx context.Context, source model.StreamSource, callbackURL string) error
}
