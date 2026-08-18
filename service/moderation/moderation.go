package moderation

import (
	"github.com/EmissarySocial/emissary/config"
)

// ReportRequest is the ActivityPub-native input for a moderation report,
// derived from an inbound Flag activity.
// See https://www.w3.org/TR/activitystreams-vocabulary/#dfn-flag
type ReportRequest struct {
	ActorID       string // reporter's ActivityPub actor URI (Flag.actor)
	ObjectID      string // URI of the flagged content (Flag.object.id)
	ObjectContent string // content text of the flagged object (if known)
	AuthorID      string // author of the flagged content (if known)
	Comment       string // report comment (Flag.content)
}

// Moderation is the interface for external moderation backends.
type Moderation interface {
	SubmitReport(report ReportRequest) error
	VerifySignature(signature string, body []byte) bool
}

// NewModeration returns the appropriate Moderation implementation based on the domain config.
func NewModeration(m config.Moderation) Moderation {
	switch m.Provider {

	case config.ModerationProviderCoop:
		coop := NewCoop(m.URL, m.Coop.APIKey, m.Coop.WebhookPublicKey, m.Coop.UserItemTypeID, m.Coop.StatusItemTypeID)
		return &coop

	default:
		return Null{}
	}
}
