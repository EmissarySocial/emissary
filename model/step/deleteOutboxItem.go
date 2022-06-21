package step

import "github.com/benpate/rosetta/maps"

// DeleteOutboxItem represents an action-step that can remove a user's activity from their outbox
type DeleteOutboxItem struct{}

// NewDeleteOutboxItem returns a fully populated DeleteOutboxItem object
func NewDeleteOutboxItem(stepInfo maps.Map) (DeleteOutboxItem, error) {
	return DeleteOutboxItem{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step DeleteOutboxItem) AmStep() {}
