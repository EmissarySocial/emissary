package content

import (
	"context"
	"io"

	"github.com/EmissarySocial/emissary/model"
)

// Adapter reads content from one kind of remote source
type Adapter interface {

	// Protocol returns the StreamSource Method that this Adapter reads
	Protocol() string

	// Version returns the validator that the remote source offers for its content now, or an
	// empty string when the source offers none
	Version(ctx context.Context, source model.StreamSource) (string, error)

	// Fetch returns the content that the remote source holds now.  The version argument names
	// what the caller expects, and a source that cannot serve a named version ignores it.
	Fetch(ctx context.Context, source model.StreamSource, version string) (Item, error)

	// FetchFile opens one file that the source's content links to.  The reader fails once more
	// than maxBytes have been read, and the caller closes it.
	FetchFile(ctx context.Context, address string, maxBytes int64) (io.ReadCloser, error)

	// Subscribe asks the remote source to push change notifications to a callback URL, or returns
	// a NotImplemented error when the source cannot push
	Subscribe(ctx context.Context, source model.StreamSource, callbackURL string) error
}
