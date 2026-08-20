package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamWidget is a single Widget placed into a Stream.  The embedded Journal
// completes the data.Object interface, so that build pipelines can treat a
// StreamWidget as a first-class object.  A StreamWidget is still saved only as
// part of its containing Stream, never on its own.
type StreamWidget struct {
	StreamWidgetID primitive.ObjectID `bson:"streamWidgetId"`
	Type           string             `bson:"type"`
	Location       string             `bson:"location"`
	Label          string             `bson:"label"`
	Data           mapof.Any          `bson:"data"`

	journal.Journal `json:"-" bson:",inline"`

	// These values are not stored in the database, but injected during building
	Stream *Stream `bson:"-"`
	Widget Widget  `bson:"-"`
}

// NewStreamWidget returns a fully initialized StreamWidget of the provided type
func NewStreamWidget(widgetType string, label string, location string) StreamWidget {
	return StreamWidget{
		StreamWidgetID: primitive.NewObjectID(),
		Type:           widgetType,
		Location:       location,
		Label:          label,
		Data:           mapof.NewAny(),
	}
}

// ID returns the string representation of the StreamWidgetID
// This method satisfies the set.Value interface
func (widget StreamWidget) ID() string {
	return widget.StreamWidgetID.Hex()
}

// IsNew returns TRUE if this StreamWidget has not been assigned an ID yet
func (widget StreamWidget) IsNew() bool {
	return widget.StreamWidgetID.IsZero()
}
