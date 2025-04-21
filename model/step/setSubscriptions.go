package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetSubscriptions represents an action that can edit a top-level folder in the Domain
type SetSubscriptions struct {
	Title string
}

// NewSetSubscriptions returns a fully parsed SetSubscriptions object
func NewSetSubscriptions(stepInfo mapof.Any) (SetSubscriptions, error) {

	return SetSubscriptions{
		Title: first(stepInfo.GetString("title"), "Subscription Settings"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetSubscriptions) AmStep() {}
