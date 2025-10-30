package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

type Tag struct {
	Type string `json:"type" bson:"type"`
	Href string `json:"href" bson:"href"`
	Name string `json:"name" bson:"name"`
}

func NewTag(value any) Tag {
	var object mapof.Any = convert.MapOfAny(value)

	return Tag{
		Type: object.GetString(vocab.PropertyType),
		Href: object.GetString(vocab.PropertyHref),
		Name: object.GetString(vocab.PropertyName),
	}
}

// Get returns a value of the given property
func (tag Tag) Get(name string) property.Value {

	switch name {

	case "type":
		return property.String(tag.Type)

	case "href":
		return property.String(tag.Href)

	case "name":
		return property.String(tag.Name)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (tag Tag) Set(name string, value any) property.Value {

	switch name {

	case "type":
		tag.Type = convert.String(value)

	case "href":
		tag.Href = convert.String(value)

	case "name":
		tag.Name = convert.String(value)
	}

	return tag
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (tag Tag) Head() property.Value {
	return tag
}

// Tail returns all values in a slice except the first
func (tag Tag) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (tag Tag) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (tag Tag) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (tag Tag) Map() map[string]any {

	return map[string]any{
		vocab.PropertyType: tag.Type,
		vocab.PropertyHref: tag.Href,
		vocab.PropertyName: tag.Name,
	}
}

// Raw returns the raw, unwrapped value being stored
func (tag Tag) Raw() any {
	return tag
}

// Clone returns a deep copy of a value
func (tag Tag) Clone() property.Value {
	return tag
}
