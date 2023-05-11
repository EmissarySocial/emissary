package model

import (
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func StreamWidgetSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamWidgetId": schema.String{Format: "objectId", Required: true},
			"type":           schema.String{Required: true},
			"data": schema.Object{
				Wildcard: schema.Any{},
			},
		},
	}
}

func (x StreamWidget) GetStringOK(key string) (string, bool) {

	switch key {

	case "streamWidgetId":
		return convert.String(x.StreamWidgetID), true

	case "type":
		return convert.String(x.Type), true
	}

	return "", false
}

func (x *StreamWidget) SetString(key string, value string) bool {

	switch key {

	case "streamWidgetId":
		if streamWidgetID, err := primitive.ObjectIDFromHex(value); err == nil {
			x.StreamWidgetID = streamWidgetID
		}

	case "type":
		x.Type = value
		return true
	}

	return false
}

func (x *StreamWidget) GetPointer(key string) (any, bool) {

	switch key {

	case "data":
		return &(x.Data), true
	}

	return nil, false
}
