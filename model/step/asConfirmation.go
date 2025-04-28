package step

import (
	"github.com/benpate/rosetta/mapof"
)

// AsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type AsConfirmation struct {
	Icon    string
	Title   string
	Message string
	Submit  string
}

// NewAsConfirmation returns a fully initialized AsConfirmation object
func NewAsConfirmation(stepInfo mapof.Any) (AsConfirmation, error) {

	return AsConfirmation{
		Icon:    stepInfo.GetString("icon"),
		Title:   stepInfo.GetString("title"),
		Message: stepInfo.GetString("message"),
		Submit:  first(stepInfo.GetString("submit"), "Continue"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step AsConfirmation) AmStep() {}
