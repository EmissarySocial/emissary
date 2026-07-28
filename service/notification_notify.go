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
	"github.com/benpate/turbine/queue"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// objectSummaryMaxLength caps the length of the plain-text ObjectSummary snapshot
const objectSummaryMaxLength = 200

/******************************************
 * Central Detection Hook
 ******************************************/

// NotifyFromActivity creates a Notification whenever an inbound activity mentions, replies to, or
// reacts to the recipient User's content
func (service *Notification) NotifyFromActivity(session data.Session, user *model.User, activity streams.Document) error {

	// Called centrally on the inbound path (handler.activitypub_user.PostInbox) for EVERY activity,
	// regardless of Following state -- the Mastodon "NotifyService" pattern.  FOLLOW is the
	// exception, created in inbox_follow_any.go once the Accept is sent (see NotifyFollow).

	// A notification failure must never fail the inbox request, so callers derp.Report and continue.

	// RULE: Never notify a user about their own actions.
	if activity.ActorID() == user.ActivityPubURL() {
		return nil
	}

	switch activity.Type() {

	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate:
		return service.notifyFromCreateOrUpdate(session, user, activity)

	case vocab.ActivityTypeLike, vocab.ActivityTypeDislike, vocab.ActivityTypeAnnounce:
		// The reacted-to object must be a local Stream owned by this user.
		if stream := model.NewStream(); service.loadLocalStreamOwnedBy(session, activity.Object().ID(), user, &stream) {
			return service.NotifyReaction(session, user, activity, &stream)
		}

	case vocab.ActivityTypeUndo, vocab.ActivityTypeDelete:
		return service.removeForActivity(session, user, activity)
	}

	return nil
}

// notifyFromCreateOrUpdate handles the Create/Update branch of NotifyFromActivity, classifying the
// activity down a strict precedence ladder: DIRECT, then REPLY, then MENTION.  Exactly one
// Notification is created per activity.
func (service *Notification) notifyFromCreateOrUpdate(session data.Session, user *model.User, activity streams.Document) error {

	object := activity.Object()

	// DIRECT: a private message, which outranks everything below it.  A DM that also replies to
	// one of this user's posts is still a DM -- it lives in the Conversations app, and that is
	// where its Notification must point.
	if isDirectMessage(activity, user) {
		return service.NotifyDirectMessage(session, user, activity, object)
	}

	// REPLY: the object's inReplyTo resolves to a local Stream owned by this user.
	// Replies are classified FIRST — a MENTION specifically means a mention that is NOT
	// a reply.  We never create both a MENTION and a REPLY for the same activity.
	// Only for Create — an Update to an existing reply should not create a new record;
	// it falls through to the mention branch, where dedup (by activityID) refreshes the
	// existing record in place, preserving its Type.
	if activity.Type() == vocab.ActivityTypeCreate {
		if inReplyTo := object.InReplyTo().ID(); inReplyTo != "" {
			if stream := model.NewStream(); service.loadLocalStreamOwnedBy(session, inReplyTo, user, &stream) {
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

// NotifyDirectMessage creates a DIRECT notification for the recipient User: a private message that
// belongs to the Conversations app rather than the public ActivityStream viewer.
func (service *Notification) NotifyDirectMessage(session data.Session, user *model.User, activity streams.Document, object streams.Document) error {

	notification := service.newNotification(user, model.NotificationTypeDirect, activity)
	notification.ObjectURL = object.ID()
	notification.InReplyTo = object.InReplyTo().ID()

	// A DIRECT Notification's Subtype is the message's CODEC, not the recipient's follow-state
	// (see the Subtype note in model/notification_constants.go).  It is stamped here rather than in
	// notify(), which stamps follow-state for every other type.
	notification.Subtype = messageCodec(object)

	// RULE: never snapshot MLS ciphertext as a display summary.  The object's content is an opaque
	// base64 blob that only the recipient's Conversations client can decrypt, so storing it would
	// put noise in the notification list AND on the lock screen (the Web Push body is this field).
	if notification.Subtype != model.NotificationSubtypeMLS {
		notification.ObjectSummary = objectSummary(object)
	}

	return service.notify(session, user, activity, &notification)
}

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

// notificationMatchKeys returns the rule keys an activity's notification is judged by: the delivering
// actor's identity keys plus the unwrapped payload's own keys (author, hashtags). Wrapper activities
// carry their tags on the inner object, so the activity is unwrapped before reading content keys; a
// link-shaped (bare string) object contributes nothing, because reading it would fetch.
func notificationMatchKeys(activity streams.Document) []string {
	return append(model.ActorMatchKeys(activity.ActorID()), model.DocumentMatchKeys(activity.UnwrapActivity())...)
}

// notify applies the Rule filter, dedups against any existing notification for the same activity,
// stamps the actor's follow-state subtype, applies the recipient's channel settings, saves the
// record, and (if the notification surfaces) publishes an SSE nudge and enqueues a Web Push task.
func (service *Notification) notify(session data.Session, user *model.User, activity streams.Document, notification *model.Notification) error {

	const location = "service.Notification.notify"

	// RULE: a blocked OR muted actor -- or a payload carrying a blocked/muted hashtag (D12) --
	// generates no notifications (R9/R16). Unlike the wire gate, MUTE counts here -- a muted actor
	// is silent. LABEL never suppresses; labels attach at render (Phase 6).
	disposition, err := service.ruleService.DispositionForKeys(session, user.UserID, notificationMatchKeys(activity), time.Now().Unix())

	if err != nil {
		return derp.Wrap(err, location, "Checking notification rules", user.UserID, activity.ActorID())
	}

	if disposition.IsFiltered() {
		return nil
	}

	// Dedup: if we already have a notification for this activity, refresh its display snapshot
	// instead of inserting a duplicate (handles Update-after-Create).
	existing, err := service.LoadOrCreate(session, user.UserID, notification.ActivityID)

	if err != nil {
		return derp.Wrap(err, location, "Loading/create notification", user.UserID, notification.ActivityID)
	}

	isNew := existing.CreateDate == 0

	if isNew {
		// Follow-state is a fact about receipt time; stamp it once, on new records only.
		//
		// RULE: DIRECT is exempt.  Its Subtype carries the message's codec (MLS/PLAINTEXT), stamped
		// by NotifyDirectMessage, and overwriting it here would erase the one fact that tells the
		// UI it cannot render the content.
		if notification.Type != model.NotificationTypeDirect {
			notification.Subtype = service.subtypeFor(session, user.UserID, notification.Actor.ProfileURL)
		}
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
	// through the normal retention purge.  They remain in the list as passive history.
	if isNew && !surfaced {
		notification.ReadDate = time.Now().Unix()
	}

	// Save the notification
	if err := service.Save(session, notification, "NotifyFromActivity"); err != nil {
		return derp.Wrap(err, location, "Saving notification", notification)
	}

	if !surfaced {
		return nil
	}

	// Publish an in-app SSE nudge (best-effort, published post-commit).
	service.publishSSE(session, notification.UserID)

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
			return derp.Wrap(err, location, "Deleting FOLLOW notification by actor", user.UserID, activity.ActorID())
		}
		return nil
	}

	// Primary match: delete by the reversed activity's ID.
	if activityID := object.ID(); activityID != "" {
		if err := service.DeleteByActivityID(session, user.UserID, activityID, "undo"); err != nil {
			return derp.Wrap(err, location, "Deleting notification by activityId", user.UserID, activityID)
		}
	}

	// Delete/Tombstone of a post also retracts any mention/reply notifications that pointed at it.
	if activity.Type() == vocab.ActivityTypeDelete {
		if objectURL := object.ID(); objectURL != "" {
			if err := service.DeleteByObjectURL(session, user.UserID, objectURL, "delete"); err != nil {
				return derp.Wrap(err, location, "Deleting notification by objectUrl", user.UserID, objectURL)
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

	derp.Report(derp.Wrap(err, location, "Loading Following record", userID, actorProfileURL))
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

// publishSSE sends a best-effort in-app realtime nudge for a new notification.  It rides the
// post-commit spool (like enqueueWebPush) so the nudge cannot fire before the Notification row
// is committed and visible to the badge/list queries it triggers.  queue.WithInline() keeps the
// task in-process — the realtime broker holds live sockets, so a stored or cross-process run
// would nudge nobody.
func (service *Notification) publishSSE(session data.Session, userID primitive.ObjectID) {
	postcommit.Publish(session, service.queue, "PublishRealtimeMessage", mapof.Any{
		"hostname": uri.Hostname(service.host),
		"objectId": userID.Hex(),
		"topic":    realtime.TopicNotification,
	}, queue.WithInline())
}

// enqueueWebPush enqueues a best-effort Web Push delivery task for a new notification.
// Published post-commit: the task references this Notification record, so it must not run
// until the enclosing transaction has committed.
func (service *Notification) enqueueWebPush(session data.Session, notification *model.Notification) {
	postcommit.Publish(session, service.queue, "SendWebPushNotification", mapof.Any{
		"hostname":       uri.Hostname(service.host),
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

// isDirectMessage returns TRUE if this activity is a private message TO this user: non-public, and
// addressed to them BY NAME.
//
// Both halves are load-bearing.  Non-public alone would also match a followers-only post, which is
// a timeline post that belongs in the public viewer, not a conversation.  The distinction is exact
// rather than heuristic: a followers-only post addresses the author's *followers collection* URL,
// while a direct message addresses this user's *actor* URL, so only a real DM names them.
//
// Addressing -- not the presence of a Mention tag -- is the test, because a DM that never tags the
// recipient is still a DM, and it already appears in their Conversations app (the direct-message
// inbox collection serves every non-public activity).  Before this, such a message produced an
// unread badge in the chat app and no Notification at all.
//
// Recipients from the activity AND its object are both consulted: the conversations-mls codecs put
// `to` on both, but other clients address only one or the other.  A union is safe here because this
// is a classification, not an access gate.
func isDirectMessage(activity streams.Document, user *model.User) bool {

	if activity.IsPublic() {
		return false
	}

	actorURL := user.ActivityPubURL()

	if actorURL == "" {
		return false
	}

	if activity.Recipients().Contains(actorURL) {
		return true
	}

	return activity.Object().Recipients().Contains(actorURL)
}

// messageCodec returns the Subtype for a DIRECT Notification: MLS when the message is end-to-end
// encrypted ciphertext, otherwise PLAINTEXT.  This mirrors the media-type test in
// handler/activitypub.IsMLSCreate (duplicated rather than imported, because `service` must not
// depend on `handler`); the privacy and inline-object conditions that function also checks are
// already established by isDirectMessage and by reading the object we were handed.
func messageCodec(object streams.Document) string {

	if object.MediaType() == vocab.MediaTypeMLS {
		return model.NotificationSubtypeMLS
	}

	return model.NotificationSubtypePlaintext
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

	if runes := []rune(stripped); len(runes) > objectSummaryMaxLength {
		return strings.TrimSpace(string(runes[:objectSummaryMaxLength])) + "…"
	}

	return stripped
}
