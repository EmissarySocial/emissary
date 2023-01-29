package activitystreams

import "github.com/benpate/derp"

type ContextList []Context

func NewContextList(capacity int) ContextList {
	return make(ContextList, 0, capacity)
}

func (contextList *ContextList) UnmarshalJSON(data []byte) error {
	return derp.NewInternalError("activitystreams.ContextList.MarshalJSON", "Not implemented", nil)
}

func (contextList ContextList) MarshalJSON() ([]byte, error) {
	return []byte{}, derp.NewInternalError("activitystreams.ContextList.MarshalJSON", "Not implemented", nil)
}
