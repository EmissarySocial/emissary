package model

import (
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StreamWidget struct {
	StreamWidgetID primitive.ObjectID `json:"streamWidgetId" bson:"streamWidgetId"`
	Type           string             `json:"type"           bson:"type"`
	Location       string             `json:"location"       bson:"location"`
	Label          string             `json:"label"          bson:"label"`
	Data           mapof.Any          `json:"data"           bson:"data"`

	// These values are not stored in the database, but injected during rendering
	Stream *Stream `json:"-" bson:"-"`
	Widget Widget  `json:"-" bson:"-"`
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
