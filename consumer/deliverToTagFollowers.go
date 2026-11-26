package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DeliverToTagFollowers adds a public post to the home timeline of every User who follows one of its #hashtags
func DeliverToTagFollowers(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.DeliverToTagFollowers"

	// Collect arguments (a missing authorId is fine: remote posts have no local author)
	url := args.GetString("url")
	authorID, _ := primitive.ObjectIDFromHex(args.GetString("authorId"))

	// Load the post (almost always from the cache, since it was just received or published)
	document, err := factory.ActivityStream().AppClient().Load(url)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Loading document", url))
	}

	// RULE: Only public posts reach people who don't follow the author (unlisted ones stay off tag timelines)
	if !service.IsPublicPost(document) {
		return queue.Success()
	}

	// Find everyone who follows any of the post's tags
	follows, err := factory.TagFollow().QueryByValues(session, service.DocumentTagValues(document))

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Querying tag follows", url))
	}

	// Add the post to each follower's timeline, once per follower
	followingService := factory.Following()
	delivered := map[primitive.ObjectID]bool{}

	for _, follow := range follows {

		// RULE: Authors don't need their own posts, and a User following two of the post's tags gets it once
		if follow.UserID == authorID || delivered[follow.UserID] {
			continue
		}

		delivered[follow.UserID] = true

		// One User's failure shouldn't keep the post from everyone else
		if err := followingService.SaveTaggedNewsItem(session, follow.UserID, follow.Name, document); err != nil {
			derp.Report(derp.Wrap(err, location, "Saving news item", follow.UserID, url))
		}
	}

	// Hashtags, delivered fresh
	return queue.Success()
}
