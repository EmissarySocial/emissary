package asstrict

import (
	"time"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
)

type Object struct {
	Type         string
	ID           string
	URL          string
	Actor        ActorSummary
	AttributedTo ActorSummary
	InReplyTo    string
	Replies      string
	Name         string
	Context      string
	Summary      string
	Content      string
	Published    time.Time
	Tag          Slice[Tag]
	Attachment   Slice[Attachment]
	Image        Image
	Icon         Image
}

// Get returns a value of the given property
func (object Object) Get(name string) property.Value {

	switch name {

	case vocab.PropertyType:
		return property.String(object.Type)

	case vocab.PropertyID:
		return property.String(object.ID)

	case vocab.PropertyURL:
		return property.String(object.URL)

	case vocab.PropertyActor:
		return object.Actor

	case vocab.PropertyAttributedTo:
		return object.AttributedTo

	case vocab.PropertyInReplyTo:
		return property.String(object.InReplyTo)

	case vocab.PropertyReplies:
		return property.String(object.Replies)

	case vocab.PropertyName:
		return property.String(object.Name)

	case vocab.PropertyContext:
		return property.String(object.Context)

	case vocab.PropertySummary:
		return property.String(object.Summary)

	case vocab.PropertyContent:
		return property.String(object.Content)

	case vocab.PropertyPublished:
		return property.Time(object.Published)

	case vocab.PropertyTag:
		return object.Tag

	case vocab.PropertyAttachment:
		return object.Attachment

	case vocab.PropertyImage:
		return object.Image

	case vocab.PropertyIcon:
		return object.Icon
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (object Object) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyType:
		object.Type = convert.String(value)

	case vocab.PropertyID:
		object.ID = convert.String(value)

	case vocab.PropertyURL:
		object.URL = convert.String(value)

	case vocab.PropertyActor:
		object.Actor = NewActorSummary(value)

	case vocab.PropertyAttributedTo:
		object.AttributedTo = NewActorSummary(value)

	case vocab.PropertyInReplyTo:
		object.InReplyTo = convert.String(value)

	case vocab.PropertyReplies:
		object.Replies = convert.String(value)

	case vocab.PropertyName:
		object.Name = convert.String(value)

	case vocab.PropertyContext:
		object.Context = convert.String(value)

	case vocab.PropertySummary:
		object.Summary = convert.String(value)

	case vocab.PropertyContent:
		object.Content = convert.String(value)

	case vocab.PropertyPublished:
		object.Published = convert.Time(value)

	case vocab.PropertyTag:
		object.Tag = NewSlice(NewTag, value)

	case vocab.PropertyAttachment:
		object.Attachment = NewSlice(NewAttachment, value)

	case vocab.PropertyImage:
		object.Image = NewImage(value)

	case vocab.PropertyIcon:
		object.Icon = NewImage(value)
	}

	return object
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (object Object) Head() property.Value {
	return object
}

// Tail returns all values in a slice except the first
func (object Object) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (object Object) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (object Object) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (object Object) Map() map[string]any {

	return map[string]any{
		vocab.PropertyType:         object.Type,
		vocab.PropertyID:           object.ID,
		vocab.PropertyURL:          object.URL,
		vocab.PropertyActor:        object.Actor.Map(),
		vocab.PropertyAttributedTo: object.AttributedTo,
		vocab.PropertyInReplyTo:    object.InReplyTo,
		vocab.PropertyReplies:      object.Replies,
		vocab.PropertyName:         object.Name,
		vocab.PropertyContext:      object.Context,
		vocab.PropertySummary:      object.Summary,
		vocab.PropertyContent:      object.Content,
		vocab.PropertyPublished:    object.Published,
		vocab.PropertyTag:          object.Tag.Map(),
		vocab.PropertyAttachment:   object.Attachment.Map(),
		vocab.PropertyImage:        object.Image.Map(),
		vocab.PropertyIcon:         object.Icon.Map(),
	}
}

// Raw returns the raw, unwrapped value being stored
func (object Object) Raw() any {
	return object
}

// Clone returns a deep copy of a value
func (object Object) Clone() property.Value {
	return object
}
