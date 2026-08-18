package moderation

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// Null is a no-op Moderation implementation used when no provider is configured.
type Null struct{}

func (n Null) SubmitReport(auth model.Authorization, report txn.PostReport) (object.Report, error) {
	return object.Report{}, nil
}

func (n Null) VerifySignature(signature string, body []byte) bool {
	return false
}
