package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewsItemSchema returns a JSON Schema that describes this object
func NewsItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"newsItemId":  schema.String{Format: "objectId"},
			"userId":      schema.String{Format: "objectId"},
			"followingId": schema.String{Format: "objectId"},
			"folderId":    schema.String{Format: "objectId"},
			"socialRole":  schema.String{MaxLength: 64},
			"origin":      OriginLinkSchema(),
			"references":  schema.Array{Items: OriginLinkSchema()},
			"url":         schema.String{Format: "url"},
			"inReplyTo":   schema.String{Format: "url"},
			"response": schema.Object{
				Properties: schema.ElementMap{
					vocab.ActivityTypeAnnounce: schema.String{Format: "objectId"},
					vocab.ActivityTypeLike:     schema.String{Format: "objectId"},
					vocab.ActivityTypeDislike:  schema.String{Format: "objectId"},
				},
			},
			"stateId":     schema.String{Enum: []string{NewsItemStateUnread, NewsItemStateRead, NewsItemStateMuted, NewsItemStateNewReplies}},
			"publishDate": schema.Integer{BitSize: 64},
			"readDate":    schema.Integer{BitSize: 64},
			"rank":        schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Methods
 ******************************************/

func (newsItem *NewsItem) GetPointer(name string) (any, bool) {
	switch name {

	case "socialRole":
		return &newsItem.SocialRole, true

	case "origin":
		return &newsItem.Origin, true

	case "references":
		return &newsItem.References, true

	case "url":
		return &newsItem.URL, true

	case "inReplyTo":
		return &newsItem.InReplyTo, true

	case "response":
		return &newsItem.Response, true

	case "stateId":
		return &newsItem.StateID, true

	case "publishDate":
		return &newsItem.PublishDate, true

	case "readDate":
		return &newsItem.ReadDate, true

	case "rank":
		return &newsItem.Rank, true

	default:
		return nil, false
	}
}

func (newsItem *NewsItem) GetStringOK(name string) (string, bool) {

	switch name {

	case "newsItemId":
		return newsItem.NewsItemID.Hex(), true

	case "userId":
		return newsItem.UserID.Hex(), true

	case "followingId":
		return newsItem.FollowingID.Hex(), true

	case "folderId":
		return newsItem.FolderID.Hex(), true

	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

func (newsItem *NewsItem) SetString(name string, value string) bool {

	switch name {

	case "newsItemId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			newsItem.NewsItemID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			newsItem.UserID = objectID
			return true
		}

	case "followingId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			newsItem.FollowingID = objectID
			return true
		}

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			newsItem.FolderID = objectID
			return true
		}
	}

	return false
}
