package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamSchema returns the JSON-Schema that validates a Stream
func StreamSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"streamId":         schema.String{Format: "objectId", Required: true},
			"parentId":         schema.String{Format: "objectId"},
			"parentIds":        schema.Array{Items: schema.String{Format: "objectId"}},
			"rank":             schema.Integer{Minimum: null.NewInt64(0)},
			"rankAlt":          schema.Integer{Minimum: null.NewInt64(0)},
			"token":            schema.String{Format: "token", MaxLength: 64},
			"navigationId":     schema.String{MaxLength: 256},
			"templateId":       schema.String{Format: "token", MaxLength: 128, Required: true},
			"parentTemplateId": schema.String{Format: "token", MaxLength: 128},
			"socialRole":       schema.String{Format: "token", MaxLength: 128},
			"stateId":          schema.String{Format: "token", MaxLength: 128},
			"groups":           permissionSchema(),
			"circles":          permissionSchema(),
			"products":         permissionSchema(),
			"url":              schema.String{Format: "url"},
			"name":             schema.String{Format: "text", MaxLength: 256},
			"label":            schema.String{Format: "text", MaxLength: 256},
			"summary":          schema.String{Format: "html", MaxLength: 2048},
			"icon":             schema.String{Format: "token", MaxLength: 64},
			"iconUrl":          schema.String{Format: "url"},
			"attributedTo":     PersonLinkSchema(),
			"context":          schema.String{MaxLength: 2048},
			"inReplyTo":        schema.String{Format: "uri"},
			"content":          ContentSchema(),
			"widgets":          WidgetSchema(),
			"startDate":        datetime.Schema(),
			"endDate":          datetime.Schema(),
			"hashtags":         schema.Array{Items: schema.String{Format: "token", MaxLength: 32}},
			"tags":             TagListSchema(),
			"location":         geo.AddressSchema(),
			"data":             schema.Object{Wildcard: schema.Any{}},
			"publishDate":      schema.Integer{BitSize: 64},
			"unpublishDate":    schema.Integer{BitSize: 64},
			"isPublished":      schema.Boolean{},
			"isFeatured":       schema.Boolean{},
			"replyCount":       schema.Integer{Minimum: null.NewInt64(0)},
			"likeCount":        schema.Integer{Minimum: null.NewInt64(0)},
			"dislikeCount":     schema.Integer{Minimum: null.NewInt64(0)},
			"shareCount":       schema.Integer{Minimum: null.NewInt64(0)},
			"syndication":      schema.Array{Items: schema.String{Required: true, Format: "token", MaxLength: 64}},
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

// permissionSchema defines the schema for the three separate
// permission maps: groups, circles, and products.
func permissionSchema() schema.Element {
	return schema.Object{
		Wildcard: schema.Array{
			Items: schema.String{Format: "objectId"},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

// GetPointer returns a pointer to the named field, and TRUE if the name is recognized
// It is part of the schema.PointerGetter interface
func (stream *Stream) GetPointer(name string) (any, bool) {

	switch name { // NOSONAR: There really are this many properties to check..

	case "parentIds":
		return &stream.ParentIDs, true

	case "groups":
		return &stream.Groups, true

	case "circles":
		return &stream.Circles, true

	case "products":
		return &stream.Products, true

	case "url":
		return &stream.URL, true

	case "name":
		return &stream.Label, true

	case "label":
		return &stream.Label, true

	case "summary":
		return &stream.Summary, true

	case "icon":
		return &stream.Icon, true

	case "iconUrl":
		return &stream.IconURL, true

	case "content":
		return &stream.Content, true

	case "widgets":
		return &stream.Widgets, true

	case "location":
		return &stream.Location, true

	case "data":
		return &stream.Data, true

	case "hashtags":
		return &stream.Hashtags, true

	case "tags":
		return &stream.Tags, true

	case "attributedTo":
		return &stream.AttributedTo, true

	case "inReplyTo":
		return &stream.InReplyTo, true

	case "context":
		return &stream.Context, true

	case "rank":
		return &stream.Rank, true

	case "rankAlt":
		return &stream.RankAlt, true

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

	case "isFeatured":
		return &stream.IsFeatured, true

	case "replyCount":
		return &stream.ReplyCount, true

	case "likeCount":
		return &stream.LikeCount, true

	case "dislikeCount":
		return &stream.DislikeCount, true

	case "shareCount":
		return &stream.ShareCount, true

	case "startDate":
		return &stream.StartDate, true

	case "endDate":
		return &stream.EndDate, true

	case "syndication":
		return &stream.Syndication, true

	default:
		return nil, false
	}
}

// GetBoolOK returns the named boolean value, and TRUE if the name is recognized
// It is part of the schema.BoolGetter interface
func (stream *Stream) GetBoolOK(name string) (bool, bool) {

	switch name {

	// This is a READ-ONLY, virtual field that's computed based on Publish and UnPublish dates.
	case "isPublished":
		return stream.IsPublished(), true
	}

	return false, false
}

// SetBool writes the named boolean value, and returns TRUE if the name is recognized
// It is part of the schema.BoolSetter interface
func (stream *Stream) SetBool(name string, value bool) bool {

	switch name {

	// This is a READ-ONLY, Virtual field.  To prevent errors, we're going to
	// lie and say we set the value, but not actually change anything.
	case "isPublished":
		return true
	}

	return false
}

// GetStringOK returns the named string value, and TRUE if the name is recognized
// It is part of the schema.StringGetter interface
func (stream *Stream) GetStringOK(name string) (string, bool) {

	switch name {

	case "streamId":
		return stream.StreamID.Hex(), true

	case "parentId":
		return stream.ParentID.Hex(), true

	case "startDate":
		return stream.StartDate.String(), true

	case "endDate":
		return stream.EndDate.String(), true

	default:
		return "", false
	}
}

// SetString writes the named string value, and returns TRUE if the name is recognized
// It is part of the schema.StringSetter interface
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
	case "isFeatured":
		stream.IsFeatured = convert.Bool(value)
		return true

	case "startDate":
		if dateTime, ok := convert.TimeOk(value, time.Time{}); ok {
			stream.StartDate.Time = dateTime
			return true
		}

	case "endDate":
		if dateTime, ok := convert.TimeOk(value, time.Time{}); ok {
			stream.EndDate.Time = dateTime
			return true
		}
	}

	return false
}
