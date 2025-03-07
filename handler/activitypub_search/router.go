package activitypub_search

import (
	"github.com/benpate/hannibal/inbox"
)

// inboxRouter defines the package-level router for search/ActivityPub requests
var inboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()
