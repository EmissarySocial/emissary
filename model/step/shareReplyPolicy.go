package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ShareReplyPolicy is a Step that stamps a Stream's ReplyPolicy on the Streams in its User's outbox
// that use one Template, such as a Notes collection passing its policy to the User's notes.
type ShareReplyPolicy struct {
	TemplateID string
}

// NewShareReplyPolicy returns a fully initialized ShareReplyPolicy step, or an error if its configuration is invalid
func NewShareReplyPolicy(stepInfo mapof.Any) (ShareReplyPolicy, error) {

	templateID := stepInfo.GetString("template")

	if templateID == "" {
		return ShareReplyPolicy{}, derp.Internal("model.step.NewShareReplyPolicy", "Missing 'template'", stepInfo)
	}

	return ShareReplyPolicy{
		TemplateID: templateID,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ShareReplyPolicy) Name() string {
	return "share-reply-policy"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ShareReplyPolicy) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ShareReplyPolicy) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ShareReplyPolicy) RequiredRoles() []string {
	return []string{}
}
