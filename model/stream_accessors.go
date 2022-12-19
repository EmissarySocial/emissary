package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter Methods
 *********************************/

func (stream *Stream) GetBool(name string) bool {
	switch name {
	case "asFeature":
		return stream.AsFeature
	default:
		return false
	}
}

func (stream *Stream) GetInt(name string) int {
	switch name {
	case "rank":
		return stream.Rank
	default:
		return 0
	}
}

func (stream *Stream) GetInt64(name string) int64 {
	switch name {
	case "publishDate":
		return stream.PublishDate
	case "unpublishDate":
		return stream.UnPublishDate
	default:
		return 0
	}
}

func (stream *Stream) GetFloat(name string) float64 {
	return 0
}

func (stream *Stream) GetString(name string) string {
	switch name {
	case "token":
		return stream.Token
	case "topLevelId":
		return stream.TopLevelID
	case "templateId":
		return stream.TemplateID
	case "stateId":
		return stream.StateID
	default:
		return ""
	}
}

func (stream *Stream) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "streamId":
		return stream.StreamID
	case "parentId":
		return stream.ParentID
	default:
		return primitive.NilObjectID
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (stream *Stream) SetBool(name string, value bool) bool {
	switch name {
	case "asFeature":
		stream.AsFeature = value
		return true
	default:
		return false
	}
}

func (stream *Stream) SetInt(name string, value int) bool {
	switch name {
	case "rank":
		stream.Rank = value
		return true
	default:
		return false
	}
}

func (stream *Stream) SetInt64(name string, value int64) bool {
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

func (stream *Stream) SetFloat(name string, value float64) bool {
	return false
}

func (stream *Stream) SetString(name string, value string) bool {
	switch name {
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
	default:
		return false
	}
}

func (stream *Stream) SetObjectID(name string, value primitive.ObjectID) bool {
	switch name {
	case "streamId":
		stream.StreamID = value
		return true
	case "parentId":
		stream.ParentID = value
		return true
	default:
		return false
	}
}

/*********************************
 * Tree Traversal Methods
 *********************************/

func (stream *Stream) GetChild(name string) (any, bool) {
	switch name {
	case "permissions":
		return &stream.Permissions, true
	case "defaultAllow":
		return &stream.DefaultAllow, true
	case "document":
		return &stream.Document, true
	case "inReplyTo":
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
