package moderation

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// Moderation is the interface for external moderation backends.
type Moderation interface {
	SubmitReport(auth model.Authorization, report txn.PostReport) (object.Report, error)
	VerifySignature(signature string, body []byte) bool
}

// NewModeration returns the appropriate Moderation implementation based on the domain config.
func NewModeration(m config.Moderation) Moderation {
	switch m.Provider {

	case config.ModerationProviderCoop:
		coop := NewCoop(m.URL, m.Coop.APIKey, m.Coop.WebhookPublicKey)
		return &coop

	default:
		return Null{}
	}
}
