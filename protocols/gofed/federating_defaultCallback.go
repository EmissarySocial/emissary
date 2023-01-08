package gofed

import (
	"context"

	"github.com/go-fed/activity/pub"
)

// DefaultCallback is called for types that the library can deserialize but is not
// handled by the application's callbacks returned in the FederatingCallbacks method.
func (fed Federating) DefaultCallback(c context.Context, activity pub.Activity) error {
	return nil
}
