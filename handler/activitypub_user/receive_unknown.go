package activitypub_user

import (
	"github.com/benpate/hannibal/streams"
	"github.com/davecgh/go-spew/spew"
)

func receive_Unknown(context Context, activity streams.Document) error {
	spew.Dump("RECEIVED UNRECOGNIZED ACTIVITYPUB MESSAGE", activity.Value())
	return nil
}
