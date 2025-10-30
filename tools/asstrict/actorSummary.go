package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

type ActorSummary struct {
	Type              string     `json:"type" bson:"type"`
	ID                string     `json:"id" bson:"id"`
	Name              string     `json:"name" bson:"name"`
	PreferredUsername string     `json:"preferredUsername" bson:"preferredUsername"`
	Summary           string     `json:"summary" bson:"summary"`
	Image             Image      `json:"image" bson:"image"`
	Icon              Image      `json:"icon" bson:"icon"`
	Tag               Slice[Tag] `json:"tag" bson:"tag"`
	URL               string     `json:"url" bson:"url"`
}

func NewActorSummary(value any) ActorSummary {

	var actorSummary mapof.Any = convert.MapOfAny(value)

	return ActorSummary{
		Type:              actorSummary.GetString(vocab.PropertyType),
		ID:                actorSummary.GetString(vocab.PropertyID),
		Name:              actorSummary.GetString(vocab.PropertyName),
		PreferredUsername: actorSummary.GetString(vocab.PropertyPreferredUsername),
		Summary:           actorSummary.GetString(vocab.PropertySummary),
		Image:             NewImage(actorSummary.GetMap(vocab.PropertyImage)),
		Icon:              NewImage(actorSummary.GetMap(vocab.PropertyIcon)),
		Tag:               NewSlice(NewTag, actorSummary.GetSliceOfMap(vocab.PropertyTag)),
		URL:               actorSummary.GetString(vocab.PropertyURL),
	}

}

// Get returns a value of the given property
func (actorSummary ActorSummary) Get(name string) property.Value {

	switch name {

	case vocab.PropertyType:
		return property.String(actorSummary.Type)

	case vocab.PropertyID:
		return property.String(actorSummary.ID)

	case vocab.PropertyName:
		return property.String(actorSummary.Name)

	case vocab.PropertyPreferredUsername:
		return property.String(actorSummary.PreferredUsername)

	case vocab.PropertySummary:
		return property.String(actorSummary.Summary)

	case vocab.PropertyImage:
		return actorSummary.Image

	case vocab.PropertyIcon:
		return actorSummary.Icon

	case vocab.PropertyTag:
		return actorSummary.Tag

	case vocab.PropertyURL:
		return property.String(actorSummary.URL)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (actorSummary ActorSummary) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyType:
		actorSummary.Type = convert.String(value)

	case vocab.PropertyID:
		actorSummary.ID = convert.String(value)

	case vocab.PropertyName:
		actorSummary.Name = convert.String(value)

	case vocab.PropertyPreferredUsername:
		actorSummary.PreferredUsername = convert.String(value)

	case vocab.PropertySummary:
		actorSummary.Summary = convert.String(value)

	case vocab.PropertyImage:
		actorSummary.Image = NewImage(value)

	case vocab.PropertyIcon:
		actorSummary.Icon = NewImage(value)

	case vocab.PropertyTag:
		actorSummary.Tag = NewSlice(NewTag, value)

	case vocab.PropertyURL:
		actorSummary.URL = convert.String(value)
	}

	return actorSummary
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (actorSummary ActorSummary) Head() property.Value {
	return actorSummary
}

// Tail returns all values in a slice except the first
func (actorSummary ActorSummary) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (actorSummary ActorSummary) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (actorSummary ActorSummary) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (actorSummary ActorSummary) Map() map[string]any {

	return map[string]any{
		vocab.PropertyType:              actorSummary.Type,
		vocab.PropertyID:                actorSummary.ID,
		vocab.PropertyName:              actorSummary.Name,
		vocab.PropertyPreferredUsername: actorSummary.PreferredUsername,
		vocab.PropertySummary:           actorSummary.Summary,
		vocab.PropertyImage:             actorSummary.Image.Map(),
		vocab.PropertyIcon:              actorSummary.Icon.Map(),
		vocab.PropertyTag:               actorSummary.Tag.SliceOfMap(),
		vocab.PropertyURL:               actorSummary.URL,
	}
}

// Raw returns the raw, unwrapped value being stored
func (actorSummary ActorSummary) Raw() any {
	return actorSummary
}

// Clone returns a deep copy of a value
func (actorSummary ActorSummary) Clone() property.Value {
	return actorSummary
}
