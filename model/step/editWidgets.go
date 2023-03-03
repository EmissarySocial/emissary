package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// EditWidgets represents an action-step that can edit/update Container in a streamDraft.
type EditWidgets struct {
	Filename string
}

func NewEditWidgets(stepInfo mapof.Any) (EditWidgets, error) {

	return EditWidgets{
		Filename: first.String(stepInfo.GetString("filename"), "edit-widgets"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditWidgets) AmStep() {}
