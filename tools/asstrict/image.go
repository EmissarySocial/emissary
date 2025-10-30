package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

type Image struct {
	Type      string
	Width     int
	Height    int
	Href      string
	MediaType string
	BlurHash  string
}

func NewImage(value any) Image {

	var object mapof.Any = convert.MapOfAny(value)

	return Image{
		Type:      object.GetString(vocab.PropertyType),
		Width:     object.GetInt(vocab.PropertyWidth),
		Height:    object.GetInt(vocab.PropertyHeight),
		Href:      object.GetString(vocab.PropertyHref),
		MediaType: object.GetString(vocab.PropertyMediaType),
		BlurHash:  object.GetString(vocab.PropertyBlurHash),
	}
}

// Get returns a value of the given property
func (image Image) Get(name string) property.Value {

	switch name {

	case vocab.PropertyType:
		return property.String(image.Type)

	case vocab.PropertyWidth:
		return property.Int(image.Width)

	case vocab.PropertyHeight:
		return property.Int(image.Height)

	case vocab.PropertyHref:
		return property.String(image.Href)

	case vocab.PropertyMediaType:
		return property.String(image.MediaType)

	case vocab.PropertyBlurHash:
		return property.String(image.BlurHash)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (image Image) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyType:
		image.Type = convert.String(value)

	case vocab.PropertyWidth:
		image.Width = convert.Int(value)

	case vocab.PropertyHeight:
		image.Height = convert.Int(value)

	case vocab.PropertyHref:
		image.Href = convert.String(value)

	case vocab.PropertyMediaType:
		image.MediaType = convert.String(value)

	case vocab.PropertyBlurHash:
		image.BlurHash = convert.String(value)
	}

	return image
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (image Image) Head() property.Value {
	return image
}

// Tail returns all values in a slice except the first
func (image Image) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (image Image) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (image Image) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (image Image) Map() map[string]any {

	return map[string]any{}
}

// Raw returns the raw, unwrapped value being stored
func (image Image) Raw() any {
	return image
}

// Clone returns a deep copy of a value
func (image Image) Clone() property.Value {
	return image
}
