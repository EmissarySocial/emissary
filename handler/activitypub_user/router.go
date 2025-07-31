package activitypub_user

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/inbox"
)

var inboxRouter inbox.Router[Context] = inbox.NewRouter[Context]()

type Context struct {
	factory *domain.Factory
	session data.Session
	user    *model.User
}
