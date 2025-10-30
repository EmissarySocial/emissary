package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
)

type Activity struct {
	ID     string
	Type   string
	Actor  string
	Object string
}

// Get returns a value of the given property
func (activity Activity) Get(name string) property.Value {

	switch name {

	case vocab.PropertyID:
		return property.String(activity.ID)

	case vocab.PropertyType:
		return property.String(activity.Type)

	case vocab.PropertyActor:
		return property.String(activity.Actor)

	case vocab.PropertyObject:
		return property.String(activity.Object)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (activity Activity) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyID:
		activity.ID = convert.String(value)

	case vocab.PropertyType:
		activity.Type = convert.String(value)

	case vocab.PropertyActor:
		activity.Actor = convert.String(value)

	case vocab.PropertyObject:
		activity.Object = convert.String(value)
	}

	return activity
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (activity Activity) Head() property.Value {
	return activity
}

// Tail returns all values in a slice except the first
func (activity Activity) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (activity Activity) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (activity Activity) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (activity Activity) Map() map[string]any {

	return map[string]any{
		vocab.PropertyID:     activity.ID,
		vocab.PropertyType:   activity.Type,
		vocab.PropertyActor:  activity.Actor,
		vocab.PropertyObject: activity.Object,
	}
}

// Raw returns the raw, unwrapped value being stored
func (activity Activity) Raw() any {
	return activity
}

// Clone returns a deep copy of a value
func (activity Activity) Clone() property.Value {
	return activity
}
