package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetProducts represents an action that can edit a top-level folder in the Domain
type SetProducts struct {
	Title string
}

// NewSetProducts returns a fully parsed SetProducts object
func NewSetProducts(stepInfo mapof.Any) (SetProducts, error) {

	return SetProducts{
		Title: first(stepInfo.GetString("title"), "Product Settings"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetProducts) AmStep() {}
