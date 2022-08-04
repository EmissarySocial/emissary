package step

import (
	"github.com/benpate/rosetta/maps"
)

// SendMentions represents an action-step that forwards the user to a new page.
type SendMentions struct{}

// NewSendMentions returns a fully initialized SendMentions object
func NewSendMentions(stepInfo maps.Map) (SendMentions, error) {
	return SendMentions{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SendMentions) AmStep() {}
