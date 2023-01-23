package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// EditContent represents an action-step that can edit/update Container in a streamDraft.
type EditContent struct {
	Filename string
	Format   string
}

func NewEditContent(stepInfo mapof.Any) (EditContent, error) {

	return EditContent{
		Filename: first.String(stepInfo.GetString("file"), stepInfo.GetString("actionId")),
		Format:   first.String(stepInfo.GetString("format"), "editorjs"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditContent) AmStep() {}
