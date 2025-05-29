package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithMerchantAccount is a Step that returns a new Follower Builder
type WithMerchantAccount struct {
	SubSteps []Step
}

// NewNewWithMerchantAccount returns a fully initialized NewWithMerchantAccount object
func NewWithMerchantAccount(stepInfo mapof.Any) (WithMerchantAccount, error) {

	const location = "NewNewWithMerchantAccount"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithMerchantAccount{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithMerchantAccount{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithMerchantAccount) Name() string {
	return "with-merchant-account"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithMerchantAccount) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithMerchantAccount) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
