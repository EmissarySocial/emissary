package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func StreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamId":         schema.String{Format: "objectId"},
			"parentId":         schema.String{Format: "objectId"},
			"parentIds":        schema.Array{Items: schema.String{Format: "objectId"}},
			"rank":             schema.Integer{Minimum: null.NewInt64(0)},
			"token":            schema.String{Format: "token", MaxLength: 128},
			"navigationId":     schema.String{},
			"templateId":       schema.String{MaxLength: 128},
			"parentTemplateId": schema.String{MaxLength: 128},
			"socialRole":       schema.String{MaxLength: 128},
			"stateId":          schema.String{MaxLength: 128},
			"permissions":      PermissionSchema(),
			"defaultAllow":     schema.Array{Items: schema.String{Format: "objectId"}},
			"url":              schema.String{Format: "url"},
			"label":            schema.String{MaxLength: 128},
			"summary":          schema.String{MaxLength: 2048},
			"imageUrl":         schema.String{Format: "url"},
			"attributedTo":     PersonLinkSchema(),
			"context":          schema.String{Format: "url"},
			"inReplyTo":        schema.String{Format: "url"},
			"content":          ContentSchema(),
			"widgets":          WidgetSchema(),
			"tags":             schema.Object{Wildcard: schema.String{}},
			"data":             schema.Object{Wildcard: schema.Any{}},
			"publishDate":      schema.Integer{BitSize: 64},
			"unpublishDate":    schema.Integer{BitSize: 64},
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
 * Getter/Setter Interfaces
 *********************************/

func (stream *Stream) GetPointer(name string) (any, bool) {

	switch name {

	case "parentIds":
		return &stream.ParentIDs, true

	case "permissions":
		return &stream.Permissions, true

	case "defaultAllow":
		return &stream.DefaultAllow, true

	case "url":
		return &stream.URL, true

	case "label":
		return &stream.Label, true

	case "summary":
		return &stream.Summary, true

	case "imageUrl":
		return &stream.ImageURL, true

	case "content":
		return &stream.Content, true

	case "widgets":
		return &stream.Widgets, true

	case "data":
		return &stream.Data, true

	case "tags":
		return &stream.Tags, true

	case "attributedTo":
		return &stream.AttributedTo, true

	case "inReplyTo":
		return &stream.InReplyTo, true

	case "rank":
		return &stream.Rank, true

	case "publishDate":
		return &stream.PublishDate, true

	case "unpublishDate":
		return &stream.UnPublishDate, true

	case "socialRole":
		return &stream.SocialRole, true

	case "navigationId":
		return &stream.NavigationID, true

	case "stateId":
		return &stream.StateID, true

	case "templateId":
		return &stream.TemplateID, true

	case "parentTemplateId":
		return &stream.TemplateID, true

	case "token":
		return &stream.Token, true

	default:
		return nil, false
	}
}

func (stream *Stream) GetStringOK(name string) (string, bool) {

	switch name {

	case "streamId":
		return stream.StreamID.Hex(), true

	case "parentId":
		return stream.ParentID.Hex(), true

	default:
		return "", false
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
	}

	return false
}
