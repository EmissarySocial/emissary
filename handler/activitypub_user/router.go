package activitypub_user

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/inbox"
)

var inboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()

var outboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()

type Context struct {
	factory *service.Factory
	session data.Session
	user    *model.User
}
