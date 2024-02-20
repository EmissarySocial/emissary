package activitypub_user

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/pub"
)

var inboxRouter pub.Router[Context]

type Context struct {
	factory *domain.Factory
	user    *model.User
}
