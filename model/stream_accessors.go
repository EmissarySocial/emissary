package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Sortable Methods
 *********************************/

func (stream *Stream) GetSort(fieldName string) any {
	switch fieldName {
	case "publishDate":
		return stream.PublishDate
	case "document.label":
		return stream.Document.Label
	case "rank":
		return stream.Rank
	default:
		return 0
	}
}

/*********************************
 * Schema Getter Interfaces
 *********************************/

func (stream *Stream) GetBoolOK(name string) (bool, bool) {
	switch name {
	case "asFeature":
		return stream.AsFeature, true
	default:
		return false, false
	}
}

func (stream *Stream) GetIntOK(name string) (int, bool) {
	switch name {
	case "rank":
		return stream.Rank, true
	default:
		return 0, false
	}
}

func (stream *Stream) GetInt64OK(name string) (int64, bool) {
	switch name {
	case "publishDate":
		return stream.PublishDate, true
	case "unpublishDate":
		return stream.UnPublishDate, true
	default:
		return 0, false
	}
}

func (stream *Stream) GetStringOK(name string) (string, bool) {
	switch name {

	case "streamId":
		return stream.StreamID.Hex(), true
	case "parentId":
		return stream.ParentID.Hex(), true
	case "token":
		return stream.Token, true
	case "topLevelId":
		return stream.TopLevelID, true
	case "templateId":
		return stream.TemplateID, true
	case "stateId":
		return stream.StateID, true
	default:
		return "", false
	}
}

/*********************************
 * Schema Setter Interfaces
 *********************************/

func (stream *Stream) SetBoolOK(name string, value bool) bool {
	switch name {
	case "asFeature":
		stream.AsFeature = value
		return true
	default:
		return false
	}
}

func (stream *Stream) SetIntOK(name string, value int) bool {
	switch name {
	case "rank":
		stream.Rank = value
		return true
	default:
		return false
	}
}

func (stream *Stream) SetInt64OK(name string, value int64) bool {
	switch name {
	case "publishDate":
		stream.PublishDate = value
		return true
	case "unpublishDate":
		stream.UnPublishDate = value
		return true
	default:
		return false
	}
}

func (stream *Stream) SetStringOK(name string, value string) bool {
	switch name {

	case "streamId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			stream.StreamID = objectID
			return true
		}

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			stream.ParentID = objectID
			return true
		}

	case "token":
		stream.Token = value
		return true

	case "topLevelId":
		stream.TopLevelID = value
		return true

	case "templateId":
		stream.TemplateID = value
		return true

	case "stateId":
		stream.StateID = value
		return true
	}

	return false
}

/*********************************
 * Tree Traversal Methods
 *********************************/

func (stream *Stream) GetObjectOK(name string) (any, bool) {

	switch name {

	case "permissions":
		return &stream.Permissions, true

	case "defaultAllow":
		return &stream.DefaultAllow, true

	case "document":
		return &stream.Document, true

	case "replyTo":
		return &stream.InReplyTo, true

	case "origin":
		return &stream.Origin, true

	case "content":
		return &stream.Content, true

	case "data":
		return &stream.Data, true

	default:
		return nil, false
	}
}
