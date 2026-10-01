package mastodon

import (
	"math"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

const (
	reactorDefaultLimit = 40 // Mastodon's documented default page size
	reactorMaxLimit     = 80 // Mastodon's documented maximum
)

// GetStatus_RebloggedBy lists the accounts that boosted a post, newest first.
// https://docs.joinmastodon.org/methods/statuses/#reblogged_by
func GetStatus_RebloggedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_RebloggedBy) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetStatus_RebloggedBy) ([]object.Account, toot.PageInfo, error) {
		return listReactors(serverFactory, auth, t.Host, t.ID, t.MaxID, t.Limit, vocab.ActivityTypeAnnounce, "handler.mastodon.GetStatus_RebloggedBy")
	}
}

// GetStatus_FavouritedBy lists the accounts that liked a post, newest first.
// https://docs.joinmastodon.org/methods/statuses/#favourited_by
func GetStatus_FavouritedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_FavouritedBy) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetStatus_FavouritedBy) ([]object.Account, toot.PageInfo, error) {
		return listReactors(serverFactory, auth, t.Host, t.ID, t.MaxID, t.Limit, vocab.ActivityTypeLike, "handler.mastodon.GetStatus_FavouritedBy")
	}
}

// listReactors lists the accounts that made one type of response (a like or a boost) to a post.
func listReactors(serverFactory *server.Factory, auth model.Authorization, host string, statusID string, maxID string, limit int64, responseType string, location string) ([]object.Account, toot.PageInfo, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return nil, toot.PageInfo{}, err
	}

	defer cancel()

	postURL, err := reactionTargetURL(factory, session, auth, statusID)

	if err != nil {
		return nil, toot.PageInfo{}, derp.Wrap(err, location, "Finding post", statusID)
	}

	// Responses are paged by their create date, which also serves as the cursor
	maxDate := int64(math.MaxInt64)

	if parsed, err := strconv.ParseInt(maxID, 10, 64); err == nil {
		maxDate = parsed
	}

	responses, err := factory.Response().QueryByObjectAndDate(session, postURL, responseType, maxDate, reactorLimit(limit))

	if err != nil {
		return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying responses", postURL)
	}

	actors := make([]actorRef, 0, len(responses))
	seen := make(map[string]bool, len(responses))

	for _, response := range responses {

		if response.Actor == "" || seen[response.Actor] {
			continue
		}

		seen[response.Actor] = true
		actors = append(actors, actorRef{url: response.Actor, since: time.UnixMilli(response.CreateDate)})
	}

	pageInfo := toot.PageInfo{}

	if length := len(responses); length > 0 {
		pageInfo.MaxID = strconv.FormatInt(responses[length-1].CreateDate, 10)
		pageInfo.MinID = strconv.FormatInt(responses[0].CreateDate, 10)
	}

	return actorsToAccounts(factory, session, auth, actors), pageInfo, nil
}

// reactionTargetURL returns the URL of the post behind a status ID, provided the caller may view it.
func reactionTargetURL(factory *service.Factory, session data.Session, auth model.Authorization, statusID string) (string, error) {

	stream := model.NewStream()

	if err := loadStreamByStatusID(factory, session, statusID, &stream); err == nil {

		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			return "", derp.Wrap(err, "handler.mastodon.reactionTargetURL", "Viewing stream", statusID)
		}

		return stream.ActivityPubURL(), nil
	}

	return resolveStatusURL(factory, session, auth, statusID), nil
}

// reactorLimit clamps a client's page size to the range Mastodon documents.
func reactorLimit(limit int64) int {

	if limit <= 0 {
		return reactorDefaultLimit
	}

	return int(min(limit, reactorMaxLimit))
}
