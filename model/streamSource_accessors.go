package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSourceSchema returns a validating schema for StreamSource objects
func StreamSourceSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamSourceId": schema.String{Format: "objectId"},
			"streamId":       schema.String{Format: "objectId", Required: true},
			"method":         schema.String{Enum: []string{StreamSourceMethodHTTPS}, Required: true},
			"url":            schema.String{Format: "url", Required: true, MaxLength: 2048},
			"config":         schema.Object{Wildcard: schema.String{MaxLength: 1024}},
			"version":        schema.String{MaxLength: 256},
			"contentHash":    schema.String{MaxLength: 64},
			"status":         schema.String{Enum: []string{StreamSourceStatusNew, StreamSourceStatusLoading, StreamSourceStatusSuccess, StreamSourceStatusFailure}},
			"statusMessage":  schema.String{Format: "text", MaxLength: 1024},
			"lastSynced":     schema.Integer{Minimum: null.NewInt64(0), BitSize: 64},
			"syncNow":        schema.Boolean{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (streamSource *StreamSource) GetPointer(name string) (any, bool) {

	switch name {

	case "method":
		return &streamSource.Method, true

	case "url":
		return &streamSource.URL, true

	case "config":
		return &streamSource.Config, true

	case "version":
		return &streamSource.Version, true

	case "contentHash":
		return &streamSource.ContentHash, true

	case "status":
		return &streamSource.Status, true

	case "statusMessage":
		return &streamSource.StatusMessage, true

	case "lastSynced":
		return &streamSource.LastSynced, true

	case "syncNow":
		return &streamSource.SyncNow, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (streamSource StreamSource) GetStringOK(name string) (string, bool) {

	switch name {

	case "streamSourceId":
		return streamSource.StreamSourceID.Hex(), true

	case "streamId":
		return streamSource.StreamID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetString writes the named property. Implements schema.StringSetter.
func (streamSource *StreamSource) SetString(name string, value string) bool {

	switch name {

	case "streamSourceId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			streamSource.StreamSourceID = objectID
			return true
		}

	case "streamId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			streamSource.StreamID = objectID
			return true
		}
	}

	return false
}
