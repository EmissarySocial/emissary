package activitypub_application

import (
	"github.com/benpate/hannibal/router"
)

// inboxRouter routes activities sent to the @application inbox
var inboxRouter = router.New[Context]()
