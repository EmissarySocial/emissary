package consumer

import (
	"context"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleCleanup applies a Rule change retroactively (R8: blocks pause, then clean up). The args are
// a SNAPSHOT of the rule at commit time -- under hard delete (D16) the row may already be gone --
// and every pass re-derives its targets with the engine's own key comparison, so nothing on the
// hot path maintains bookkeeping. All passes read local data only; cleanup never fetches remotely
// (D7), least of all from a just-blocked domain.
func RuleCleanup(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.RuleCleanup"

	log.Trace().Msg("Task: Rule-Cleanup")

	// Parse the snapshot
	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args))
	}

	matchKey := args.GetString("matchKey")

	if matchKey == "" {
		return queue.Failure(derp.Internal(location, "Missing matchKey in Rule-Cleanup args", args))
	}

	// RULE: an ADMIN rule (no owning User) fans out as one batched task per User (R8)
	if userID.IsZero() {
		return ruleCleanup_fanOutToUsers(factory, session, args)
	}

	// Purge pass: newsfeed items go for both BLOCK and MUTE transitions (D1)
	if args.GetBool("purgeAll") || args.GetBool("purgeNewsfeed") {
		if err := ruleCleanup_newsfeed(factory, session, userID, args); err != nil {
			return queue.Error(derp.Wrap(err, location, "Purging NewsItems"))
		}
	}

	// BLOCK-only passes: notifications, collection entries, and relationship pausing
	if args.GetBool("purgeAll") {

		if err := ruleCleanup_notifications(factory, session, userID, matchKey); err != nil {
			return queue.Error(derp.Wrap(err, location, "Purging Notifications"))
		}

		if err := ruleCleanup_collectionItems(factory, session, userID, matchKey); err != nil {
			return queue.Error(derp.Wrap(err, location, "Purging CollectionItems"))
		}

		if err := ruleCleanup_pauseRelationships(factory, session, userID, matchKey); err != nil {
			return queue.Error(derp.Wrap(err, location, "Pausing relationships"))
		}
	}

	// Restore pass: a deleted (or re-aimed) BLOCK re-evaluates every paused Follower
	if args.GetBool("restore") {
		if err := ruleCleanup_restoreFollowers(factory, session, userID); err != nil {
			return queue.Error(derp.Wrap(err, location, "Restoring paused Followers"))
		}
	}

	// My work here is done. To the Batmobile!
	return queue.Success()
}

// ruleCleanup_fanOutToUsers re-enqueues an ADMIN-tier cleanup as one task per User, so a
// domain-wide block never runs one unbounded pass inside a single transaction.
func ruleCleanup_fanOutToUsers(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.ruleCleanup_fanOutToUsers"

	users, err := factory.User().RangeAll(session)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Ranging Users"))
	}

	for user := range users {

		perUser := mapof.Any{}

		for key, value := range args {
			perUser[key] = value
		}

		perUser["userId"] = user.UserID.Hex()
		postcommit.Publish(session, factory.Queue(), "Rule-Cleanup", perUser)
	}

	return queue.Success()
}

// ruleCleanup_newsfeed removes this User's NewsItems that the rule covers: the fast path deletes
// by origin.followingId when the snapshot carries one; the general path evaluates each item's
// origin/URL keys -- and, for TAG rules, the locally-cached document's keys (never a fetch, D7).
func ruleCleanup_newsfeed(factory *service.Factory, session data.Session, userID primitive.ObjectID, args mapof.Any) error {

	const location = "consumer.ruleCleanup_newsfeed"

	newsFeedService := factory.NewsFeed()
	matchKey := args.GetString("matchKey")

	// Fast path: every NewsItem from the rule's own Following
	if followingID, err := primitive.ObjectIDFromHex(args.GetString("followingId")); err == nil && !followingID.IsZero() {

		rangeFunc, err := newsFeedService.RangeByFollowingID(session, userID, followingID)

		if err != nil {
			return derp.Wrap(err, location, "Ranging NewsItems by FollowingID", followingID)
		}

		for newsItem := range rangeFunc {
			if err := newsFeedService.Delete(session, &newsItem, "Removed by rule"); err != nil {
				return derp.Wrap(err, location, "Deleting NewsItem", newsItem.NewsItemID)
			}
		}
	}

	// General path: evaluate every remaining NewsItem with the engine's own comparison
	rangeFunc, err := newsFeedService.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Ranging NewsItems by UserID", userID)
	}

	isTagRule := args.GetString("type") == model.RuleTypeTag

	for newsItem := range rangeFunc {

		keys := append(model.ActorMatchKeys(newsItem.Origin.URL), model.DomainMatchKeys(newsItem.URL)...)

		// TAG rules can only match through the document's hashtags, which the NewsItem does not
		// store -- so read the locally-cached document (a database read, never a fetch)
		if !slices.Contains(keys, matchKey) && isTagRule {
			if document, exists := ruleCleanup_cachedDocument(factory, newsItem.URL); exists {
				keys = append(keys, model.DocumentMatchKeys(document)...)
			}
		}

		if !slices.Contains(keys, matchKey) {
			continue
		}

		if err := newsFeedService.Delete(session, &newsItem, "Removed by rule"); err != nil {
			return derp.Wrap(err, location, "Deleting NewsItem", newsItem.NewsItemID)
		}
	}

	return nil
}

// ruleCleanup_notifications removes this User's Notifications whose snapshotted actor the rule
// covers -- the same ActorMatchKeys comparison the engine uses, never a second matcher.
func ruleCleanup_notifications(factory *service.Factory, session data.Session, userID primitive.ObjectID, matchKey string) error {

	const location = "consumer.ruleCleanup_notifications"

	notificationService := factory.Notification()
	rangeFunc, err := notificationService.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Ranging Notifications", userID)
	}

	for notification := range rangeFunc {

		if !slices.Contains(model.ActorMatchKeys(notification.Actor.ProfileURL), matchKey) {
			continue
		}

		if err := notificationService.Delete(session, &notification, "Removed by rule"); err != nil {
			return derp.Wrap(err, location, "Deleting Notification", notification.NotificationID)
		}
	}

	return nil
}

// ruleCleanup_collectionItems removes this User's response-collection entries that the rule
// covers. An entry stores only its URI, so DOMAIN rules match on the URI's host and ACTOR rules
// attribute through the local AS cache; unattributable entries are left for the render-time
// overlay backstop (D7). Removal goes through Collection.RemoveItem so totalItems stays accurate.
func ruleCleanup_collectionItems(factory *service.Factory, session data.Session, userID primitive.ObjectID, matchKey string) error {

	const location = "consumer.ruleCleanup_collectionItems"

	collectionService := factory.Collection()
	rangeFunc, err := factory.CollectionItem().Range(session, exp.Equal("userId", userID))

	if err != nil {
		return derp.Wrap(err, location, "Ranging CollectionItems", userID)
	}

	for item := range rangeFunc {

		// ACTOR rules: attribute via the locally-cached activity (a database read, never a fetch)
		if keys := model.DomainMatchKeys(item.URI); !slices.Contains(keys, matchKey) {

			document, exists := ruleCleanup_cachedDocument(factory, item.URI)

			if !exists {
				continue
			}

			keys = append(model.ActorMatchKeys(document.ActorID()), model.DocumentMatchKeys(document)...)

			if !slices.Contains(keys, matchKey) {
				continue
			}
		}

		// Remove through the Collection so `totalItems` is refreshed alongside the row
		collection := model.NewCollection()

		if err := collectionService.Load(session, exp.Equal("_id", item.CollectionID), &collection); err != nil {
			derp.Report(derp.Wrap(err, location, "Loading Collection for cleanup; skipping item", item.CollectionID))
			continue
		}

		if err := collectionService.RemoveItem(session, &collection, item.URI); err != nil {
			return derp.Wrap(err, location, "Removing CollectionItem", item.URI)
		}
	}

	return nil
}

// ruleCleanup_pauseRelationships pauses this User's relationships with the blocked actor:
// Followers move to PAUSED (delivery fan-out skips them), and Following rows send their
// Undo/Follow, stop polling, and wait for a manual re-follow (R8).
func ruleCleanup_pauseRelationships(factory *service.Factory, session data.Session, userID primitive.ObjectID, matchKey string) error {

	const location = "consumer.ruleCleanup_pauseRelationships"

	// Followers -> PAUSED
	followerService := factory.Follower()

	for follower := range followerService.RangeFollowers(session, model.FollowerTypeUser, userID) {

		if !slices.Contains(model.ActorMatchKeys(follower.Actor.ProfileURL), matchKey) {
			continue
		}

		if err := followerService.Pause(session, &follower); err != nil {
			return derp.Wrap(err, location, "Pausing Follower", follower.FollowerID)
		}
	}

	// Following -> paused (Undo/Follow sent, polling stops)
	followingService := factory.Following()
	rangeFunc, err := followingService.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Ranging Following records", userID)
	}

	for following := range rangeFunc {

		if keys := append(model.ActorMatchKeys(following.URL), model.ActorMatchKeys(following.ProfileURL)...); !slices.Contains(keys, matchKey) {
			continue
		}

		if err := followingService.Pause(session, &following); err != nil {
			return derp.Wrap(err, location, "Pausing Following", following.FollowingID)
		}
	}

	return nil
}

// ruleCleanup_restoreFollowers re-evaluates every PAUSED Follower against the REMAINING rules and
// reactivates the no-longer-blocked. Re-derivation (not a back-pointer) is what makes overlapping
// rules safe: a Follower covered by two blocks stays paused until the last one is gone. Followers
// who sent Undo(Follow) while paused were already soft-deleted (D6), so they never resurrect.
// Following rows are deliberately NOT auto-resumed -- we sent their Undo/Follow, so re-following
// is the user's one-click decision, never an automatic side effect.
func ruleCleanup_restoreFollowers(factory *service.Factory, session data.Session, userID primitive.ObjectID) error {

	const location = "consumer.ruleCleanup_restoreFollowers"

	followerService := factory.Follower()
	ruleService := factory.Rule()
	now := time.Now().Unix()

	for follower := range followerService.RangePausedByUserID(session, userID) {

		disposition, err := ruleService.DispositionForKeys(session, userID, model.ActorMatchKeys(follower.Actor.ProfileURL), now)

		// RULE: on a rules-query failure the Follower STAYS paused -- wrongly resuming delivery
		// to a blocked actor is the unrecoverable direction (the P5-2 posture).
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Checking rules for paused Follower; leaving paused", follower.FollowerID))
			continue
		}

		if disposition.IsBlocked() {
			continue
		}

		if err := followerService.Reactivate(session, &follower); err != nil {
			return derp.Wrap(err, location, "Reactivating Follower", follower.FollowerID)
		}
	}

	return nil
}

// ruleCleanup_cachedDocument reads a document from the LOCAL ActivityStream cache by URL. It
// never fetches from the network (D7) -- a cache miss simply reports the document as absent.
func ruleCleanup_cachedDocument(factory *service.Factory, url string) (streams.Document, bool) {

	values := factory.ActivityStream().Range(
		context.Background(),
		exp.Equal("urls", url),
		option.MaxRows(1),
	)

	for value := range values {
		return value.AsDocument(), true
	}

	return streams.NilDocument(), false
}
