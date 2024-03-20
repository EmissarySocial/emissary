package activitypub_stream

import (
	"github.com/benpate/hannibal/inbox"
)

// streamRouter defines the package-level router for stream/ActivityPub requests
var streamRouter inbox.Router[Context] = inbox.NewRouter[Context]()
