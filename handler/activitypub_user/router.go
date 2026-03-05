package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/steranko"
)

// inboxRouter defines all of the ActivityPub handlers for
// activities POST-ed to User's inboxes
var inboxRouter = router.New[Context]()

// outboxRouter defines all of the ActivityPub handlers for
// activities POSTED-ed to User's outboxes
var outboxRouter = router.New[Context]()

// Context defines custom data that is passed through the
// inbox/outbox routers to each handler
type Context struct {
	context *steranko.Context // The HTTP request context
	factory *service.Factory  // The service factory
	session data.Session      // The current data session
	user    *model.User       // The user whose inbox/outbox is being accessed
}
