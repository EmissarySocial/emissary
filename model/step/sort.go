package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/first"
)

// Sort represents an action-step that can update multiple records at once
type Sort struct {
	Keys    string
	Values  string
	Message string
}

func NewSort(stepInfo datatype.Map) (Sort, error) {

	return Sort{
		Keys:    first.String(stepInfo.GetString("keys"), "_id"),
		Values:  first.String(stepInfo.GetString("values"), "rank"),
		Message: stepInfo.GetString("message"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Sort) AmStep() {}
