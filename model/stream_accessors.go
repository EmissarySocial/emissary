package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func StreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamId":      schema.String{Format: "objectId"},
			"parentId":      schema.String{Format: "objectId"},
			"parentIds":     schema.Array{Items: schema.String{Format: "objectId"}},
			"rank":          schema.Integer{Minimum: null.NewInt64(0)},
			"token":         schema.String{Format: "token", MaxLength: 128},
			"navigationId":  schema.String{Format: "objectId"},
			"templateId":    schema.String{MaxLength: 128},
			"socialRole":    schema.String{MaxLength: 128},
			"stateId":       schema.String{MaxLength: 128},
			"permissions":   PermissionSchema(),
			"defaultAllow":  schema.Array{Items: schema.String{Format: "objectId"}},
			"document":      DocumentLinkSchema(),
			"inReplyTo":     DocumentLinkSchema(),
			"content":       ContentSchema(),
			"widgets":       WidgetSchema(),
			"data":          schema.Object{Wildcard: schema.Any{}},
			"publishDate":   schema.Integer{BitSize: 64},
			"unpublishDate": schema.Integer{BitSize: 64},
		},
	}
}

// WidgetSchema defines the structure for the "widgets" container.
func WidgetSchema() schema.Element {
	return schema.Object{
		Wildcard: schema.Array{
			Items: schema.String{Format: "token"},
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

	case "socialRole":
		return stream.SocialRole, true

	case "navigationId":
		return stream.NavigationID, true

	case "stateId":
		return stream.StateID, true

	case "templateId":
		return stream.TemplateID, true

	case "token":
		return stream.Token, true

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

	case "socialRole":
		stream.SocialRole = value
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

	case "parentIds":
		return &stream.ParentIDs, true

	case "permissions":
		return &stream.Permissions, true

	case "defaultAllow":
		return &stream.DefaultAllow, true

	case "document":
		return &stream.Document, true

	case "inReplyTo":
		return &stream.InReplyTo, true

	case "content":
		return &stream.Content, true

	case "widgets":
		return &stream.Widgets, true

	case "data":
		return &stream.Data, true

	default:
		return nil, false
	}
}
