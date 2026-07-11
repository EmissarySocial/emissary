package service

// PublishConfig collects the optional settings that modify a single Outbox.Publish (or
// UndoActivity) call. Zero value = default behavior (fan out to all of the Actor's followers
// plus the activity's own addressees).
type PublishConfig struct {
	// recipients, when non-nil, REPLACES the default follower fan-out with this explicit list of
	// recipient ACTOR URLs. Used for author-only delivery of reactions (Like/Dislike and their
	// Undos), where the activity should reach only the liked object's author — never the reactor's
	// followers. See COLLECTIONS-REDESIGN.md D7a/D7b. Recipients must be ACTOR URLs (only actors
	// have inboxes); an object URL is not deliverable.
	recipients []string

	// hasRecipients distinguishes "no override" (nil) from "override with an empty list" — an
	// explicit empty recipient set means "deliver to nobody but the activity's own addressees",
	// which is different from the default follower fan-out.
	hasRecipients bool
}

// PublishOption modifies a PublishConfig. Options are applied in order.
type PublishOption func(*PublishConfig)

// WithRecipients replaces the default follower fan-out with an explicit list of recipient ACTOR
// URLs. Passing it (even with zero URLs) suppresses the follower fan-out entirely. Any wrapper
// that forwards PublishOptions MUST spread them (`options...`) — a dropped spread silently
// discards this override and the activity fans out to all followers. See COLLECTIONS-REDESIGN.md D7b.
func WithRecipients(actorURLs ...string) PublishOption {
	return func(config *PublishConfig) {
		config.recipients = actorURLs
		config.hasRecipients = true
	}
}

// newPublishConfig applies the given options to a zero-value PublishConfig.
func newPublishConfig(options ...PublishOption) PublishConfig {
	result := PublishConfig{}

	for _, option := range options {
		option(&result)
	}

	return result
}
