package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/rosetta/convert"
)

type Slice[T property.Value] []T

func NewSlice[T property.Value](fn func(any) T, value any) Slice[T] {

	items := convert.SliceOfAny(value)

	result := make(Slice[T], 0, len(items))

	for index, item := range items {
		result[index] = fn(item)
	}

	return result
}

// Get returns a value of the given property
func (slice Slice[T]) Get(name string) property.Value {
	return slice.Head().Get(name)
}

// Set returns the value with the given property set
func (slice Slice[T]) Set(name string, value any) property.Value {
	return slice.SetSlice(name, value)
}

// SetSlice is a more strongly typed version of Set
func (slice Slice[T]) SetSlice(name string, value any) Slice[T] {

	if slice.IsNil() {
		return slice
	}

	head := slice.Head().Set(name, value)

	if match, isMatch := head.(T); isMatch {
		slice[0] = match
	}

	return slice
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (slice Slice[T]) Head() property.Value {
	if slice.IsNil() {
		return property.Nil{}
	}

	return slice[0]
}

// Tail returns all values in a slice except the first
func (slice Slice[T]) Tail() property.Value {
	if slice.IsNil() {
		return property.Nil{}
	}

	return slice[:1]
}

// Len returns the number of elements in the value
func (slice Slice[T]) Len() int {
	return len(slice)
}

// IsNil returns TRUE if the value is empty
func (slice Slice[T]) IsNil() bool {
	return slice.Len() == 0
}

// Map returns the map representation of this value
func (slice Slice[T]) Map() map[string]any {
	return slice.Head().Map()
}

// Raw returns the raw, unwrapped value being stored
func (slice Slice[T]) Raw() any {
	return slice
}

// Clone returns a deep copy of a value
func (slice Slice[T]) Clone() property.Value {
	result := make(Slice[T], 0, slice.Len())

	for index := range slice {

		cloned := slice[index].Clone()

		if match, isMatch := cloned.(T); isMatch {
			result[index] = match
			continue
		}

		// This should never happen
		var empty T
		result[index] = empty
	}

	return result
}

// / Custom Methods for Slice type
func (slice Slice[T]) SliceOfMap() []map[string]any {

	result := make([]map[string]any, 0, slice.Len())

	for index := range slice {
		result[index] = slice[index].Map()
	}

	return result
}
