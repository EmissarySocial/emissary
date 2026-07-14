package service

import (
	"html"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// objectSummaryMaxLength caps the length of the plain-text ObjectSummary snapshot
const objectSummaryMaxLength = 200

/******************************************
 * Central Detection Hook
 ******************************************/

// NotifyFromActivity inspects a single inbound ActivityPub activity and creates a Notification
// for the recipient User whenever the activity mentions, replies to, or reacts to their content.
// It is called centrally on the inbound path (handler.activitypub_user.PostInbox) for EVERY
// activity, regardless of Following state — this is the Mastodon "NotifyService" pattern.
//
// FOLLOW notifications are the exception: they are created from inbox_follow_any.go after the
// Follower record is saved and the Accept is sent (see NotifyFollow).
//
// A notification failure must never fail the inbox request; callers should derp.Report the error
// and continue.
func (service *Notification) NotifyFromActivity(session data.Session, user *model.User, activity streams.Document) error {

	// RULE: Never notify a user about their own actions.
	if activity.ActorID() == user.ActivityPubURL() {
		return nil
	}

	switch activity.Type() {

	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return service.notifyFromCreateOrUpdate(session, user, activity)

	case vocab.ActivityTypeLike, vocab.ActivityTypeDislike, vocab.ActivityTypeAnnounce:
		// The reacted-to object must be a local Stream owned by this user.
		stream := model.NewStream()
		if service.loadLocalStreamOwnedBy(session, activity.Object().ID(), user, &stream) {
			return service.NotifyReaction(session, user, activity, &stream)
		}

	case vocab.ActivityTypeUndo, vocab.ActivityTypeDelete:
		return service.removeForActivity(session, user, activity)
	}

	return nil
}

// notifyFromCreateOrUpdate handles the Create/Update branch of NotifyFromActivity: a REPLY when
// the object replies to a local Stream this user owns, otherwise a MENTION when it tags them.
func (service *Notification) notifyFromCreateOrUpdate(session data.Session, user *model.User, activity streams.Document) error {

	object := activity.Object()

	// REPLY: the object's inReplyTo resolves to a local Stream owned by this user.
	// Replies are classified FIRST — a MENTION specifically means a mention that is NOT
	// a reply.  We never create both a MENTION and a REPLY for the same activity.
	// Only for Create — an Update to an existing reply should not create a new record;
	// it falls through to the mention branch, where dedup (by activityID) refreshes the
	// existing record in place, preserving its Type.
	if activity.Type() == vocab.ActivityTypeCreate {
		if inReplyTo := object.InReplyTo().ID(); inReplyTo != "" {
			stream := model.NewStream()
			if service.loadLocalStreamOwnedBy(session, inReplyTo, user, &stream) {
				return service.NotifyReply(session, user, activity, object, &stream)
			}
		}
	}

	// MENTION: a tag[] entry names this user (and the object is not a reply to them).
	for href := range object.RangeMentions() {
		if service.isThisUser(href, user) {
			return service.NotifyMention(session, user, activity, object)
		}
	}

	return nil
}

/******************************************
 * Producers
 ******************************************/

// NotifyMention creates a MENTION notification for the recipient User.
func (service *Notification) NotifyMention(session data.Session, user *model.User, activity streams.Document, object streams.Document) error {

	notification := service.newNotification(user, model.NotificationTypeMention, activity)
	notification.ObjectURL = object.ID()
	notification.ObjectSummary = objectSummary(object)
	notification.InReplyTo = object.InReplyTo().ID()

	return service.notify(session, user, activity, &notification)
}

// NotifyReply creates a REPLY notification for the recipient User, tied to the local Stream replied to.
func (service *Notification) NotifyReply(session data.Session, user *model.User, activity streams.Document, object streams.Document, stream *model.Stream) error {

	notification := service.newNotification(user, model.NotificationTypeReply, activity)
	notification.ObjectURL = object.ID()
	notification.ObjectSummary = objectSummary(object)
	notification.InReplyTo = object.InReplyTo().ID()
	notification.StreamID = stream.StreamID

	return service.notify(session, user, activity, &notification)
}

// NotifyReaction creates a LIKE/DISLIKE/ANNOUNCE notification for the recipient User, tied to the
// local Stream that was reacted to.
func (service *Notification) NotifyReaction(session data.Session, user *model.User, activity streams.Document, stream *model.Stream) error {

	notification := service.newNotification(user, reactionType(activity.Type()), activity)
	notification.ObjectURL = stream.URL
	notification.ObjectSummary = plainText(stream.Label)
	notification.StreamID = stream.StreamID

	return service.notify(session, user, activity, &notification)
}

// NotifyFollow creates a FOLLOW notification for the recipient User.  It is called from
// inbox_follow_any.go after the Follower record is saved and the Accept is sent.
func (service *Notification) NotifyFollow(session data.Session, user *model.User, activity streams.Document) error {

	notification := service.newNotification(user, model.NotificationTypeFollow, activity)
	notification.ObjectURL = user.ActivityPubURL()

	return service.notify(session, user, activity, &notification)
}

/******************************************
 * Common notify() funnel
 ******************************************/

// notify applies the Rule filter, dedups against any existing notification for the same activity,
// stamps the actor's follow-state subtype, applies the recipient's channel settings, saves the
// record, and (if the notification surfaces) publishes an SSE nudge and enqueues a Web Push task.
func (service *Notification) notify(session data.Session, user *model.User, activity streams.Document, notification *model.Notification) error {

	const location = "service.Notification.notify"

	// RULE: blocked/muted actors do not generate notifications.
	ruleFilter := service.ruleService.Filter(user.UserID, WithBlocksOnly()) // nolint:scopeguard
	if ruleFilter.Disallow(session, &activity) {
		return nil
	}

	// Dedup: if we already have a notification for this activity, refresh its display snapshot
	// instead of inserting a duplicate (handles Update-after-Create).
	existing, err := service.LoadOrCreate(session, user.UserID, notification.ActivityID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load/create notification", user.UserID, notification.ActivityID)
	}

	isNew := existing.CreateDate == 0

	if isNew {
		// Follow-state is a fact about receipt time; stamp it once, on new records only.
		notification.Subtype = service.subtypeFor(session, user.UserID, notification.Actor.ProfileURL)
	} else {
		// An already-persisted record was found (it has a journal createDate) — refresh its display
		// snapshot in place and keep its existing ID, Type, Subtype, and ReadDate.
		existing.Actor = notification.Actor
		existing.ObjectURL = notification.ObjectURL
		existing.ObjectSummary = notification.ObjectSummary
		existing.InReplyTo = notification.InReplyTo
		notification = &existing
	}

	// SETTINGS: the recipient's enabled channels decide whether this notification surfaces
	// (SSE nudge, Web Push, unread dot).  Policy lives in model.Notification.Channels().
	surfaced := user.NotificationEnabled(notification.Channels())

	// Suppressed notifications are born read: they never light the dot, and they age out
	// through the normal purge of read rows.  They remain in the list as passive history.
	if isNew && !surfaced {
		notification.ReadDate = time.Now().Unix()
	}

	// Save the notification
	if err := service.Save(session, notification, "NotifyFromActivity"); err != nil {
		return derp.Wrap(err, location, "Unable to save notification", notification)
	}

	if !surfaced {
		return nil
	}

	// Publish an in-app SSE nudge (best-effort).
	service.publishSSE(notification.UserID)

	// Enqueue a Web Push delivery task (best-effort, published post-commit).
	service.enqueueWebPush(session, notification)

	return nil
}

/******************************************
 * Undo / Delete cleanup
 ******************************************/

// removeForActivity reverses a previously-created notification when its triggering activity is
// Undone or Deleted.
func (service *Notification) removeForActivity(session data.Session, user *model.User, activity streams.Document) error {

	const location = "service.Notification.removeForActivity"

	// The inner object of an Undo/Delete is the thing being reversed.
	object := activity.Object()

	// UNFOLLOW: an Undo/Follow (or Delete/Follow) is matched by the ACTOR, not the Follow
	// activity's id.  The id is often absent or synthetic, but the unfollowing actor is always
	// present on the Undo — and object.Type() reads the embedded type without a network fetch.
	if object.Type() == vocab.ActivityTypeFollow {
		if err := service.DeleteFollowByActor(session, user.UserID, activity.ActorID(), "unfollow"); err != nil {
			return derp.Wrap(err, location, "Unable to delete FOLLOW notification by actor", user.UserID, activity.ActorID())
		}
		return nil
	}

	// Primary match: delete by the reversed activity's ID.
	if activityID := object.ID(); activityID != "" {
		if err := service.DeleteByActivityID(session, user.UserID, activityID, "undo"); err != nil {
			return derp.Wrap(err, location, "Unable to delete notification by activityId", user.UserID, activityID)
		}
	}

	// Delete/Tombstone of a post also retracts any mention/reply notifications that pointed at it.
	if activity.Type() == vocab.ActivityTypeDelete {
		if objectURL := object.ID(); objectURL != "" {
			if err := service.DeleteByObjectURL(session, user.UserID, objectURL, "delete"); err != nil {
				return derp.Wrap(err, location, "Unable to delete notification by objectUrl", user.UserID, objectURL)
			}
		}
	}

	return nil
}

/******************************************
 * Helpers
 ******************************************/

// newNotification builds a base Notification for the recipient User from an inbound activity,
// populating the recipient, type, actor, and triggering activity ID.
func (service *Notification) newNotification(user *model.User, notificationType string, activity streams.Document) model.Notification {

	notification := model.NewNotification()
	notification.UserID = user.UserID
	notification.Type = notificationType
	notification.ActivityID = activity.ID()
	notification.Actor = actorPersonLink(activity.Actor())

	return notification
}

// subtypeFor returns the recipient's follow-state for the actor at receipt time:
// FOLLOWING if a Following record exists for (userID, actorProfileURL), otherwise
// NOT_FOLLOWING.  An unexpected lookup error is reported and fails open to FOLLOWING
// (matching the empty-subtype rule in model.Notification.Channels).
func (service *Notification) subtypeFor(session data.Session, userID primitive.ObjectID, actorProfileURL string) string {

	const location = "service.Notification.subtypeFor"

	following := model.NewFollowing()
	err := service.followingService.LoadByURL(session, userID, actorProfileURL, &following)

	if err == nil {
		return model.NotificationSubtypeFollowing
	}

	if derp.IsNotFound(err) {
		return model.NotificationSubtypeNotFollowing
	}

	derp.Report(derp.Wrap(err, location, "Unable to load Following record", userID, actorProfileURL))
	return model.NotificationSubtypeFollowing
}

// isThisUser returns TRUE if the provided href refers to the recipient User (by ActivityPub URL).
func (service *Notification) isThisUser(href string, user *model.User) bool {
	if href == "" {
		return false
	}
	return href == user.ActivityPubURL()
}

// loadLocalStreamOwnedBy loads the local Stream referenced by the provided URL and returns TRUE
// only if it exists AND is owned by (attributed to) the provided User.  A reaction/reply to
// another local user's stream must notify THAT user (via their own inbox delivery), not this one.
func (service *Notification) loadLocalStreamOwnedBy(session data.Session, url string, user *model.User, stream *model.Stream) bool {

	if url == "" {
		return false
	}

	if err := service.streamService.LoadByURL(session, url, stream); err != nil {
		return false // Not a resolvable local stream — not an error, just not our case.
	}

	return stream.AttributedTo.UserID == user.UserID
}

// publishSSE sends a best-effort in-app realtime nudge for a new notification.  The full realtime
// wiring (topic + client) lands in Phase 4; until then this is a no-op-safe channel send.
func (service *Notification) publishSSE(userID primitive.ObjectID) {
	if service.sseUpdateChannel == nil {
		return
	}
	service.sseUpdateChannel <- realtime.NewMessage_Notification(userID)
}

// enqueueWebPush enqueues a best-effort Web Push delivery task for a new notification.
// Published post-commit: the task references this Notification record, so it must not run
// until the enclosing transaction has committed.
func (service *Notification) enqueueWebPush(session data.Session, notification *model.Notification) {
	postcommit.Publish(session, service.queue, "SendWebPushNotification", mapof.Any{
		"host":           service.host,
		"userId":         notification.UserID.Hex(),
		"notificationId": notification.NotificationID.Hex(),
	})
}

// actorPersonLink builds a PersonLink from an actor streams.Document (same shape as follower.Actor).
func actorPersonLink(actor streams.Document) model.PersonLink {
	return model.PersonLink{
		ProfileURL:   actor.ID(),
		Name:         actor.Name(),
		Username:     actor.UsernameOrID(),
		IconURL:      actor.IconOrImage().URL(),
		InboxURL:     actor.Get("inbox").String(),
		EmailAddress: actor.Get("email").String(),
	}
}

// reactionType maps an ActivityPub activity type to a Notification reaction type.
func reactionType(activityType string) string {
	switch activityType {
	case vocab.ActivityTypeDislike:
		return model.NotificationTypeDislike
	case vocab.ActivityTypeAnnounce:
		return model.NotificationTypeAnnounce
	default:
		return model.NotificationTypeLike
	}
}

// objectSummary returns a short, plain-text display snapshot for a notification's object.
// It prefers the object's Name, falling back to a truncated, tag-stripped Content/Summary.
// The result is PLAIN TEXT ONLY (federated content is untrusted — see the security checklist).
func objectSummary(object streams.Document) string {

	if name := object.Name(); name != "" {
		return plainText(name)
	}

	if content := object.Content(); content != "" {
		return plainText(content)
	}

	return plainText(object.Summary())
}

// plainText strips all HTML tags and truncates to objectSummaryMaxLength runes.  bluemonday's
// StrictPolicy escapes entities, so we unescape afterward to store readable plain text.
func plainText(value string) string {

	stripped := html.UnescapeString(convert.SanitizeText(value))
	stripped = strings.TrimSpace(stripped)

	runes := []rune(stripped)
	if len(runes) > objectSummaryMaxLength {
		return strings.TrimSpace(string(runes[:objectSummaryMaxLength])) + "…"
	}

	return stripped
}
