package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// knownRemoteAccount builds a basic account for a remote actor the caller follows, or who follows the
// caller, for when their own server can't be reached. It returns false for anyone else.
func knownRemoteAccount(factory *service.Factory, session data.Session, auth model.Authorization, accountURL string) (object.Account, bool) {

	following := model.NewFollowing()

	if err := factory.Following().LoadByURL(session, auth.UserID, accountURL, &following); err == nil {
		return accountFromFollowing(&following), true
	}

	follower := model.NewFollower()

	if err := factory.Follower().LoadByActor(session, auth.UserID, accountURL, &follower); err == nil {
		return follower.Actor.Toot(), true
	}

	return object.Account{}, false
}

// accountFromFollowing builds a basic account from a Following record.
func accountFromFollowing(following *model.Following) object.Account {

	person := model.PersonLink{
		Name:       following.Label,
		Username:   following.Username,
		ProfileURL: following.ProfileURL,
		IconURL:    following.IconURL,
	}

	return person.Toot()
}

// knownProfileURL finds the profile URL of an "@user@domain" handle from the User's own records of
// people they follow or who follow them. It returns "" when the handle isn't one of those.
func knownProfileURL(factory *service.Factory, session data.Session, userID primitive.ObjectID, acct string) string {

	handle := "@" + acct

	following := model.NewFollowing()

	if err := factory.Following().Load(session, exp.Equal("userId", userID).AndEqual("username", handle), &following); err == nil && following.ProfileURL != "" {
		return following.ProfileURL
	}

	follower := model.NewFollower()

	if err := factory.Follower().Load(session, exp.Equal("parentId", userID).AndEqual("actor.username", handle), &follower); err == nil {
		return follower.Actor.ProfileURL
	}

	return ""
}
