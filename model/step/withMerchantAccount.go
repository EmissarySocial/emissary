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

// AmStep is here only to verify that this struct is a build pipeline step
func (step WithMerchantAccount) AmStep() {}
