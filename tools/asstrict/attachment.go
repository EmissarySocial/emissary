package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

type Attachment struct {
	Type      string
	MediaType string
	URL       string
	Height    int
	Width     int
	Content   string
}

func NewAttachment(value any) Attachment {

	var valueMap mapof.Any = convert.MapOfAny(value)

	return Attachment{
		Type:      valueMap.GetString(vocab.PropertyType),
		MediaType: valueMap.GetString(vocab.PropertyMediaType),
		URL:       valueMap.GetString(vocab.PropertyURL),
		Height:    valueMap.GetInt(vocab.PropertyHeight),
		Width:     valueMap.GetInt(vocab.PropertyWidth),
		Content:   valueMap.GetString(vocab.PropertyContent),
	}
}

// Get returns a value of the given property
func (attachment Attachment) Get(name string) property.Value {

	switch name {

	case vocab.PropertyType:
		return property.String(attachment.Type)

	case vocab.PropertyMediaType:
		return property.String(attachment.MediaType)

	case vocab.PropertyURL:
		return property.String(attachment.URL)

	case vocab.PropertyHeight:
		return property.Int(attachment.Height)

	case vocab.PropertyWidth:
		return property.Int(attachment.Width)

	case vocab.PropertyContent:
		return property.String(attachment.Content)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (attachment Attachment) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyType:
		attachment.Type = convert.String(value)

	case vocab.PropertyMediaType:
		attachment.MediaType = convert.String(value)

	case vocab.PropertyURL:
		attachment.URL = convert.String(value)

	case vocab.PropertyHeight:
		attachment.Height = convert.Int(value)

	case vocab.PropertyWidth:
		attachment.Width = convert.Int(value)

	case vocab.PropertyContent:
		attachment.Content = convert.String(value)
	}

	return attachment
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (attachment Attachment) Head() property.Value {
	return attachment
}

// Tail returns all values in a slice except the first
func (attachment Attachment) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (attachment Attachment) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (attachment Attachment) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (attachment Attachment) Map() map[string]any {

	return map[string]any{
		vocab.PropertyType:      attachment.Type,
		vocab.PropertyMediaType: attachment.MediaType,
		vocab.PropertyURL:       attachment.URL,
		vocab.PropertyHeight:    attachment.Height,
		vocab.PropertyWidth:     attachment.Width,
		vocab.PropertyContent:   attachment.Content,
	}
}

// Raw returns the raw, unwrapped value being stored
func (attachment Attachment) Raw() any {
	return attachment
}

// Clone returns a deep copy of a value
func (attachment Attachment) Clone() property.Value {
	return attachment
}
