package model

import (
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StreamWidget struct {
	StreamWidgetID primitive.ObjectID
	Type           string
	Location       string
	Label          string
	Data           mapof.Any
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
