package activitypub_search

import (
	"github.com/benpate/hannibal/router"
)

// inboxRouter defines the package-level router for search/ActivityPub requests
var inboxRouter = router.New[Context]()
