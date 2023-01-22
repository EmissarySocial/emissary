package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
)

// AsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type AsConfirmation struct {
	Title   string
	Message string
	Submit  string
}

// NewAsConfirmation returns a fully initialized AsConfirmation object
func NewAsConfirmation(stepInfo maps.Map) (AsConfirmation, error) {

	return AsConfirmation{
		Title:   getValue(stepInfo.GetString("title")),
		Message: getValue(stepInfo.GetString("message")),
		Submit:  first.String(getValue(stepInfo.GetString("submit")), "Continue"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AsConfirmation) AmStep() {}
