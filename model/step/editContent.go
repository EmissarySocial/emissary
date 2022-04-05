package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/first"
)

// EditContent represents an action-step that can edit/update Container in a streamDraft.
type EditContent struct {
	Filename string
}

func NewEditContent(stepInfo datatype.Map) (EditContent, error) {

	return EditContent{
		Filename: first.String(stepInfo.GetString("file"), stepInfo.GetString("actionId")),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step EditContent) AmStep() {}
