package mastodon

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/benpate/uri"
)

// PostTag_Follow implements the Mastodon "follow a hashtag" endpoint
func PostTag_Follow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Follow) (object.Tag, error) {

	const location = "handler.mastodon.PostTag_Follow"

	return func(auth model.Authorization, t txn.PostTag_Follow) (object.Tag, error) {

		// RULE: Only a signed-in User can follow a tag
		if !auth.IsAuthenticated() {
			return object.Tag{}, derp.Unauthorized(location, "Sign in to follow hashtags")
		}

		// Get the Domain factory and a database session
		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.Tag{}, err
		}

		defer cancel()

		// RULE: No one may follow a tag the admin has blocked
		name := service.TagFollowName(t.ID)
		blocked, err := isBlockedTag(factory, session, name)

		if err != nil {
			return object.Tag{}, derp.Wrap(err, location, "Checking tag state", name)
		}

		if blocked {
			return object.Tag{}, derp.Validation("This hashtag can't be followed on this server")
		}

		// Follow the tag
		if err := factory.TagFollow().Follow(session, auth.UserID, name); err != nil {
			return object.Tag{}, derp.Wrap(err, location, "Following tag", name)
		}

		// You're it
		return followedTag(t.Host, name, true), nil
	}
}

// PostTag_Unfollow implements the Mastodon "unfollow a hashtag" endpoint
func PostTag_Unfollow(serverFactory *server.Factory) func(model.Authorization, txn.PostTag_Unfollow) (object.Tag, error) {

	const location = "handler.mastodon.PostTag_Unfollow"

	return func(auth model.Authorization, t txn.PostTag_Unfollow) (object.Tag, error) {

		// RULE: Only a signed-in User can unfollow a tag
		if !auth.IsAuthenticated() {
			return object.Tag{}, derp.Unauthorized(location, "Sign in to unfollow hashtags")
		}

		// Get the Domain factory and a database session
		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.Tag{}, err
		}

		defer cancel()

		// Unfollow the tag
		name := service.TagFollowName(t.ID)

		if err := factory.TagFollow().Unfollow(session, auth.UserID, name); err != nil {
			return object.Tag{}, derp.Wrap(err, location, "Unfollowing tag", name)
		}

		// You're not it
		return followedTag(t.Host, name, false), nil
	}
}

// isBlockedTag returns TRUE if the admin has blocked the tag in Search Tags
func isBlockedTag(factory *service.Factory, session data.Session, name string) (bool, error) {

	searchTag := model.NewSearchTag()

	if err := factory.SearchTag().LoadByValue(session, name, &searchTag); err != nil {

		// A tag the admin never listed is not blocked
		if derp.IsNotFound(err) {
			return false, nil
		}

		return false, derp.Wrap(err, "handler.mastodon.isBlockedTag", "Loading SearchTag", name)
	}

	return searchTag.StateID == model.SearchTagStateBlocked, nil
}

// followedTag returns a Mastodon Tag for this server, marked as followed or not
func followedTag(host string, name string, following bool) object.Tag {
	return object.Tag{
		Name:      name,
		URL:       uri.GuessProtocolForHostname(host) + host + "/tags/" + url.PathEscape(name),
		History:   []object.TagHistory{},
		Following: following,
	}
}
