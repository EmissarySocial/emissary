package gofed

import (
	"context"

	"github.com/go-fed/activity/pub"
)

/*
Returns the application-specific logic needed for your application as callbacks for the library to invoke.

The library splits your applications behaviors between those specified in the ActivityPub spec, which will wrap your behaviors in wrapped, or behaviors not known in the ActivityPub spec which will be provided in other.

The pub.FederatingWrappedCallbacks returned provides a collection of default ActivityPub behaviors as defined in the specification. For more details on how to use these provided behaviors and supplement with your own business logic, see Federating Wrapped Callbacks. The zero-value is a valid value.

If instead you wish to override the default ActivityPub behaviors, such as doing nothing, then the other return value should contain a function with a signature like:

	other = []interface{}{
		// This function overrides the FederatingWrappedCallbacks-provided behavior
		func(c context.Context, create vocab.ActivityStreamsCreate) error {
			return nil
		},
	}

The above would replace the library's default behavior of creating the entry in the database upon receiving a Create activity.

If you want to handle an Activity that does not have a default behavior provided in pub.FederatingWrappedCallbacks, then specify it in other using a similar function signature.

Applications are not expected to handle every single ActivityStreams type and extension. The unhandled ones are passed to DefaultCallback.
*/
func (fed *Federating) FederatingCallbacks(c context.Context) (wrapped pub.FederatingWrappedCallbacks, other []interface{}, err error) {
	return pub.FederatingWrappedCallbacks{}, []interface{}{}, nil
}
