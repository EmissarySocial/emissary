package model

import (
	"math"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Notification is a recipient-centric record of something another actor did
// that involves a local User: mentioned them, replied to them, liked/disliked/
// announced their content, or followed them.  Modeled on Mastodon notifications.
//
// Supersedes the (now write-dead) model.Mention record.  Notifications are
// created centrally on the inbound ActivityPub path (see
// service.Notification.NotifyFromActivity) regardless of Following state.
type Notification struct {
	NotificationID primitive.ObjectID `bson:"_id"`                     // Unique ID for this Notification
	UserID         primitive.ObjectID `bson:"userId"`                  // Recipient (local User) who owns this Notification
	Type           string             `bson:"type"`                    // DIRECT/MENTION/REPLY/LIKE/DISLIKE/ANNOUNCE/FOLLOW
	Subtype        string             `bson:"subtype,omitempty"`       // Per-Type discriminant: MLS/PLAINTEXT for DIRECT, otherwise FOLLOWING/NOT_FOLLOWING (see notification_constants.go)
	Actor          PersonLink         `bson:"actor"`                   // Who did the thing
	ActivityID     string             `bson:"activityId,omitempty"`    // AP id of the triggering activity (dedup + undo)
	ObjectURL      string             `bson:"objectUrl,omitempty"`     // The thing acted on / the mentioning object
	ObjectSummary  string             `bson:"objectSummary,omitempty"` // Display snapshot (title/excerpt). PLAIN TEXT ONLY.
	StreamID       primitive.ObjectID `bson:"streamId,omitempty"`      // Local Stream involved, if any (cleanup + stream-page query)
	InReplyTo      string             `bson:"inReplyTo,omitempty"`     // For REPLY/MENTION threading into browse view
	ReadDate       int64              `bson:"readDate"`                // Unix epoch SECONDS when read (math.MaxInt64 = unread)
	Labels         metadata.LabelSet  `bson:"-" json:"-"`              // The viewer's rule verdict for the Actor, stamped at render time. Never persisted (R8: derive, don't record).

	journal.Journal `json:"-" bson:",inline"`
}

// NewNotification returns a fully initialized Notification object
func NewNotification() Notification {
	return Notification{
		NotificationID: primitive.NewObjectID(),
		Actor:          NewPersonLink(),
		ReadDate:       math.MaxInt64,
	}
}

// NotificationFields returns the standard list of fields loaded for a Notification
func NotificationFields() []string {
	return []string{"_id", "userId", "type", "subtype", "actor", "activityId", "objectUrl", "objectSummary", "streamId", "inReplyTo", "readDate", "createDate"}
}

// Fields returns the database fields required to populate a Notification
func (notification Notification) Fields() []string {
	return NotificationFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns a string representation of the Notification's unique id.
// This method implements the data.Object interface.
func (notification Notification) ID() string {
	return notification.NotificationID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Notification.
// It is part of the AccessLister interface
func (notification *Notification) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Notification
// It is part of the AccessLister interface
func (notification *Notification) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (notification *Notification) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && notification.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (notification *Notification) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(notification.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (notification *Notification) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Read-only Methods
 ******************************************/

// IsRead returns TRUE if this notification has a valid ReadDate
func (notification Notification) IsRead() bool {
	return notification.ReadDate < math.MaxInt64
}

// NotRead returns TRUE if this notification does not have a valid ReadDate
func (notification Notification) NotRead() bool {
	return notification.ReadDate == math.MaxInt64
}

// IsConversation returns TRUE if this Notification points at a PRIVATE message, which belongs to
// the Conversations app rather than to the public ActivityStream viewer.  It is the SINGLE source
// of that routing decision -- the notification list templates and the Web Push consumer both read
// it, so the two surfaces cannot drift apart.
//
// Records written before the DIRECT type existed are typed MENTION and keep opening the public
// viewer.  That is the honest fallback: they age out on the normal retention clock, and the
// Conversations app can only open a message it has synced.
func (notification Notification) IsConversation() bool {
	return notification.Type == NotificationTypeDirect
}

// IsEncrypted returns TRUE if this Notification points at an MLS-encrypted message.  Only the
// Conversations app holds the group's ratchet state, so nothing server-side can render the
// message's content -- callers must not try to display it.
func (notification Notification) IsEncrypted() bool {
	return notification.Subtype == NotificationSubtypeMLS
}

// Channels returns the settings channels that can surface this Notification.  A Notification
// surfaces (unread dot, SSE nudge, Web Push) if ANY of its channels is enabled in the
// recipient's User.NotificationChannels.  This is the SINGLE source of policy mapping
// notification facts (Type, Subtype) to user settings — no other code may derive channels.
func (notification Notification) Channels() []string {

	// An empty Subtype is treated as FOLLOWING (fail-open for the friendlier channel).
	mentionChannel := NotificationChannelMentionFollowing
	if notification.Subtype == NotificationSubtypeNotFollowing {
		mentionChannel = NotificationChannelMentionNotFollowing
	}

	switch notification.Type {

	case NotificationTypeDirect:
		// The direct-message channel is the SOLE authority here.  There is deliberately no
		// "either enables it" fallback to the mention channels (as REPLY has below), so switching
		// this toggle off means off.  It is not split by follow-state -- see the constant.
		//
		// Returning an empty slice here would mark every direct message born-read and deliver no
		// SSE nudge and no Web Push (see service.Notification.notify), so this case must never
		// fall through to the default.
		return []string{NotificationChannelDirectMessage}

	case NotificationTypeMention:
		return []string{mentionChannel}

	case NotificationTypeReply:
		// "Either enables it": replies surface via the Replies toggle OR the applicable
		// mention toggle (Mastodon-compatible senders auto-tag the parent author).
		return []string{NotificationChannelReply, mentionChannel}

	case NotificationTypeLike, NotificationTypeDislike, NotificationTypeAnnounce:
		return []string{NotificationChannelReaction}

	case NotificationTypeFollow:
		return []string{NotificationChannelFollow}
	}

	return []string{}
}

/******************************************
 * Write Methods
 ******************************************/

// SetState implements the model.StateSetter interface. It is used by HTML
// templates in the build pipeline to mark a notification read/unread.
func (notification *Notification) SetState(stateID string) {

	switch stateID {

	case "READ":
		notification.MarkRead()

	case "UNREAD":
		notification.MarkUnread()
	}
}

// MarkRead sets the ReadDate of this Notification to the current time (if not
// already set).  Returns TRUE if the value was changed.
func (notification *Notification) MarkRead() bool {

	if notification.ReadDate < math.MaxInt64 {
		return false
	}

	notification.ReadDate = time.Now().Unix()
	return true
}

// MarkUnread clears the ReadDate of this Notification.
// Returns TRUE if the value was changed.
func (notification *Notification) MarkUnread() bool {

	if notification.ReadDate == math.MaxInt64 {
		return false
	}

	notification.ReadDate = math.MaxInt64
	return true
}

/******************************************
 * Mastodon API
 ******************************************/

// GetRank returns the value used to paginate this Notification in the Mastodon API
// (createDate in milliseconds, matching queryExpression's createDate filter).
func (notification Notification) GetRank() int64 {
	return notification.CreateDate
}

// MastodonType maps this Notification's Type to the corresponding Mastodon notification type.
// DISLIKE has no Mastodon equivalent and maps to "" (callers should exclude it from the API).
func (notification Notification) MastodonType() string {
	switch notification.Type {
	case NotificationTypeDirect, NotificationTypeMention, NotificationTypeReply:
		// Mastodon has no separate notification type for direct messages -- a DM arrives as a
		// "mention" notification whose status carries visibility "direct".  Omitting DIRECT here
		// would hit the "" default and drop every DM from the Mastodon API.
		return "mention"
	case NotificationTypeLike:
		return "favourite"
	case NotificationTypeAnnounce:
		return "reblog"
	case NotificationTypeFollow:
		return "follow"
	default:
		return ""
	}
}

// Toot returns this Notification represented as a Mastodon API Notification object.
func (notification Notification) Toot() object.Notification {
	return object.Notification{
		ID:        notification.NotificationID.Hex(),
		Type:      notification.MastodonType(),
		CreatedAt: time.UnixMilli(notification.CreateDate).UTC().Format(time.RFC3339),
		Account:   notification.Actor.Toot(),
	}
}
