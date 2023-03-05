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
}

func NewStreamWidget(widgetType string, label string, location string) StreamWidget {
	return StreamWidget{
		StreamWidgetID: primitive.NewObjectID(),
		Type:           widgetType,
		Label:          label,
		Data:           mapof.NewAny(),
	}
}

func (widget StreamWidget) IsNew() bool {
	return widget.StreamWidgetID.IsZero()
}
