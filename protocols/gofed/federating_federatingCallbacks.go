package gofed

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
)

// Returns the application-specific logic needed for your application as callbacks for the library to invoke.
//
// The library splits your applications behaviors between those specified in the ActivityPub spec, which will
// wrap your behaviors in wrapped, or behaviors not known in the ActivityPub spec which will be provided in other.
//
// The pub.FederatingWrappedCallbacks returned provides a collection of default ActivityPub behaviors as defined
// in the specification. For more details on how to use these provided behaviors and supplement with your own
// business logic, see Federating Wrapped Callbacks. The zero-value is a valid value.
//
// If instead you wish to override the default ActivityPub behaviors, such as doing nothing, then the other
// return value should contain a function with a signature like:
//
//	other = []interface{}{
//		// This function overrides the FederatingWrappedCallbacks-provided behavior
//		func(c context.Context, create vocab.ActivityStreamsCreate) error {
//			return nil
//		},
//	}
//
// The above would replace the library's default behavior of creating the entry in the database upon receiving
// a Create activity.
//
// If you want to handle an Activity that does not have a default behavior provided in
// pub.FederatingWrappedCallbacks, then specify it in other using a similar function signature.
//
// Applications are not expected to handle every single ActivityStreams type and extension. The unhandled ones
// are passed to DefaultCallback.
func (fed Federating) FederatingCallbacks(c context.Context) (wrapped pub.FederatingWrappedCallbacks, other []any, err error) {

	// TODO: MEDIUM: Implement these callbacks as necessary once ActivityPub is running.
	// Research:
	// https://go-fed.org/ref/activity/pub#Federating-Wrapped-Callbacks

	wrappedCallbacks := pub.FederatingWrappedCallbacks{
		Create: func(c context.Context, create vocab.ActivityStreamsCreate) error {
			spew.Dump("FederatingCallbacks.Create", create)
			return nil
		},
		Update: func(c context.Context, update vocab.ActivityStreamsUpdate) error {
			spew.Dump("FederatingCallbacks.Update", update)
			return nil
		},
		Delete: func(c context.Context, delete vocab.ActivityStreamsDelete) error {
			spew.Dump("FederatingCallbacks.Delete", delete)
			return nil
		},
		Follow: func(c context.Context, follow vocab.ActivityStreamsFollow) error {
			spew.Dump("FederatingCallbacks.Follow", follow)
			return nil
		},
		Accept: func(c context.Context, accept vocab.ActivityStreamsAccept) error {
			spew.Dump("FederatingCallbacks.Accept", accept)
			return nil
		},
		Reject: func(c context.Context, reject vocab.ActivityStreamsReject) error {
			spew.Dump("FederatingCallbacks.Reject", reject)
			return nil
		},
		Add: func(c context.Context, add vocab.ActivityStreamsAdd) error {
			spew.Dump("FederatingCallbacks.Add", add)
			return nil
		},
		Remove: func(c context.Context, remove vocab.ActivityStreamsRemove) error {
			spew.Dump("FederaingCallbacks.Remove", remove)
			return nil
		},
		Like: func(c context.Context, like vocab.ActivityStreamsLike) error {
			spew.Dump("FederaingCallbacks.Like", like)
			return nil
		},
		Announce: func(c context.Context, announce vocab.ActivityStreamsAnnounce) error {
			spew.Dump("FederaingCallbacks.Announce", announce)
			return nil
		},
		Undo: func(c context.Context, undo vocab.ActivityStreamsUndo) error {
			spew.Dump("FederaingCallbacks.Undo", undo)
			return nil
		},
		Block: func(c context.Context, block vocab.ActivityStreamsBlock) error {
			spew.Dump("FederaingCallbacks.Block", block)
			return nil
		},
	}

	// This is used to override default behaviors.
	otherCallbacks := []any{}

	return wrappedCallbacks, otherCallbacks, nil
}
