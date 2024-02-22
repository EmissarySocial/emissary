package activitypub_user

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/inbox"
)

var inboxRouter inbox.Router[Context] = inbox.NewRouter[Context](inbox.WithNoDebug())

type Context struct {
	factory *domain.Factory
	user    *model.User
}
