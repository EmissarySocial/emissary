package model

import (
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StreamWidget struct {
	StreamWidgetID primitive.ObjectID `bson:"streamWidgetId"`
	Type           string             `bson:"type"`
	Location       string             `bson:"location"`
	Label          string             `bson:"label"`
	Data           mapof.Any          `bson:"data"`

	// These values are not stored in the database, but injected during building
	Stream *Stream `bson:"-"`
	Widget Widget  `bson:"-"`
}

func NewStreamWidget(widgetType string, label string, location string) StreamWidget {
	return StreamWidget{
		StreamWidgetID: primitive.NewObjectID(),
		Type:           widgetType,
		Label:          label,
		Data:           mapof.NewAny(),
	}
}

// ID returns the string representation of the StreamWidgetID
// This method satisfies the set.Value interface
func (widget StreamWidget) ID() string {
	return widget.StreamWidgetID.Hex()
}

func (widget StreamWidget) IsNew() bool {
	return widget.StreamWidgetID.IsZero()
}
