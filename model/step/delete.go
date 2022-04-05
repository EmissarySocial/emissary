package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/first"
)

// Delete represents an action-step that can delete a Stream from the Domain
type Delete struct {
	Title   string
	Message string
	Submit  string
}

// NewDelete returns a fully populated Delete object
func NewDelete(stepInfo datatype.Map) (Delete, error) {
	return Delete{
		Title:   first.String(stepInfo.GetString("title"), "Confirm Delete"),
		Message: first.String(stepInfo.GetString("message"), "Are you sure you want to delete this item?  There is NO UNDO."),
		Submit:  first.String(stepInfo.GetString("submit"), "Delete"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Delete) AmStep() {}
