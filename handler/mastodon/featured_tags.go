package mastodon

import (
	"strconv"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/benpate/uri"
)

/******************************************
 * Mastodon Featured Tags
 *
 * A User's featured tags are the #hashtags in their
 * profile bio, which Emissary already publishes.
 * https://docs.joinmastodon.org/methods/featured_tags/
 ******************************************/

// GetFeaturedTags implements the Mastodon "view your featured tags" endpoint
func GetFeaturedTags(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags) ([]object.FeaturedTag, error) {

	const location = "handler.mastodon.GetFeaturedTags"

	return func(auth model.Authorization, t txn.GetFeaturedTags) ([]object.FeaturedTag, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the signed-in User
		user := model.NewUser()

		if err := factory.User().LoadByID(session, auth.UserID, &user); err != nil {
			return nil, derp.Wrap(err, location, "Loading user", auth.UserID)
		}

		// Your tags, featuring you
		return featuredTags(factory, session, auth, &user), nil
	}
}

// PostFeaturedTag is the Mastodon "feature a tag" endpoint, which Emissary does not implement
func PostFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {

	return func(model.Authorization, txn.PostFeaturedTag) (object.FeaturedTag, error) {
		// Featured tags come from the profile bio, so there is nothing separate to add
		return object.FeaturedTag{}, errNotSupported("handler.mastodon.PostFeaturedTag", "Featuring a tag (add the #hashtag to your profile bio instead)")
	}
}

// DeleteFeaturedTag is the Mastodon "unfeature a tag" endpoint, which Emissary does not implement
func DeleteFeaturedTag(serverFactory *server.Factory) func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {

	return func(model.Authorization, txn.DeleteFeaturedTag) (struct{}, error) {
		return struct{}{}, errNoSuch("handler.mastodon.DeleteFeaturedTag", "Featured tag")
	}
}

// GetFeaturedTags_Suggestions implements the Mastodon "suggested featured tags" endpoint, and always returns an empty list
func GetFeaturedTags_Suggestions(serverFactory *server.Factory) func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {

	return func(model.Authorization, txn.GetFeaturedTags_Suggestions) ([]object.FeaturedTag, error) {
		return []object.FeaturedTag{}, nil
	}
}

// featuredTags returns a User's profile hashtags as Mastodon featured tags, counting the posts the caller may see
func featuredTags(factory *service.Factory, session data.Session, auth model.Authorization, user *model.User) []object.FeaturedTag {

	const location = "handler.mastodon.featuredTags"

	result := make([]object.FeaturedTag, 0, len(user.Hashtags))

	for _, name := range user.Hashtags {

		// Skip anything that isn't a usable tag name
		name = strings.TrimPrefix(name, "#")

		if name == "" {
			continue
		}

		// Count this User's posts with the tag; a failure leaves the count at zero
		count, newest, err := factory.Stream().SummarizeByUserHashtag(session, auth, user.UserID, hashtagSpellings(factory, session, name))

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Counting tagged posts", user.UserID, name))
		}

		result = append(result, featuredTag(user, name, count, newest))
	}

	// Feature presentation
	return result
}

// featuredTag returns one Mastodon featured tag for a User's hashtag
func featuredTag(user *model.User, name string, count int64, newest int64) object.FeaturedTag {

	// The tag's ID is its name, since Emissary stores no separate record
	result := object.FeaturedTag{
		ID:            name,
		Name:          name,
		URL:           model.HashtagURL(uri.Host(user.ProfileURL), user.TagURL, name),
		StatusesCount: strconv.FormatInt(count, 10),
	}

	// Clients require a URL, so fall back to the profile when the template has no tag links
	if result.URL == "" {
		result.URL = user.ProfileURL
	}

	if newest > 0 {
		result.LastStatusAt = time.Unix(newest, 0).UTC().Format("2006-01-02")
	}

	// Tag, you're it
	return result
}
