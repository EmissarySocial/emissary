package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/first"
)

// AsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type AsConfirmation struct {
	Title   string
	Message string
	Submit  string
}

// NewAsConfirmation returns a fully initialized AsConfirmation object
func NewAsConfirmation(stepInfo datatype.Map) (AsConfirmation, error) {

	return AsConfirmation{
		Title:   stepInfo.GetString("title"),
		Message: stepInfo.GetString("message"),
		Submit:  first.String(stepInfo.GetString("submit"), "Continue"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step AsConfirmation) AmStep() {}