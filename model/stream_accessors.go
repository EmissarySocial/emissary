package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func StreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamId":      schema.String{Format: "objectId"},
			"parentId":      schema.String{Format: "objectId"},
			"token":         schema.String{Format: "token"},
			"navigationId":  schema.String{Format: "objectId"},
			"templateId":    schema.String{},
			"stateId":       schema.String{},
			"permissions":   PermissionSchema(),
			"document":      DocumentLinkSchema(),
			"author":        PersonLinkSchema(),
			"replyTo":       DocumentLinkSchema(),
			"content":       ContentSchema(),
			"rank":          schema.Integer{},
			"publishDate":   schema.Integer{BitSize: 64},
			"unpublishDate": schema.Integer{BitSize: 64},
		},
	}
}

func PermissionSchema() schema.Element {

	return schema.Object{
		Wildcard: schema.Array{
			Items: schema.String{Format: "objectId"},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

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
	case "navigationId":
		return stream.NavigationID, true
	case "templateId":
		return stream.TemplateID, true
	case "stateId":
		return stream.StateID, true
	default:
		return "", false
	}
}

/*********************************
 * Setter Interfaces
 *********************************/

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

func (stream *Stream) SetString(name string, value string) bool {
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

	case "navigationId":
		stream.NavigationID = value
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
 * Tree Traversal Interfaces
 *********************************/

func (stream *Stream) GetObject(name string) (any, bool) {

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
