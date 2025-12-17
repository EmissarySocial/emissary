package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/inbox"
	"github.com/benpate/steranko"
)

var inboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()

var outboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()

type Context struct {
	context *steranko.Context
	factory *service.Factory
	session data.Session
	user    *model.User
}
