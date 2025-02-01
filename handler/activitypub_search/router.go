package activitypub_search

import (
	"github.com/benpate/hannibal/inbox"
)

// searchRouter defines the package-level router for search/ActivityPub requests
var searchRouter inbox.Router[Context] = inbox.NewRouter[Context]()
