package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter Methods
 *********************************/

func (doc *DocumentLink) GetInt64(name string) int64 {
	switch name {
	case "publishDate":
		return doc.PublishDate
	case "unpdateDate":
		return doc.UpdateDate
	default:
		return 0
	}
}

func (doc *DocumentLink) GetString(name string) string {
	switch name {
	case "internalId":
		return doc.InternalID.Hex()
	case "url":
		return doc.URL
	case "label":
		return doc.Label
	case "summary":
		return doc.Summary
	case "type":
		return doc.Type
	case "imageUrl":
		return doc.ImageURL
	default:
		return ""
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (doc *DocumentLink) SetInt64(name string, value int64) bool {
	switch name {
	case "publishDate":
		doc.PublishDate = value
		return true
	case "unpdateDate":
		doc.UpdateDate = value
		return true
	default:
		return false
	}
}

func (doc *DocumentLink) SetString(name string, value string) bool {
	switch name {
	case "internalId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			doc.InternalID = objectID
			return true
		}
	case "url":
		doc.URL = value
		return true
	case "label":
		doc.Label = value
		return true
	case "summary":
		doc.Summary = value
		return true
	case "type":
		doc.Type = value
		return true
	case "imageUrl":
		doc.ImageURL = value
		return true
	}

	return false
}

/*********************************
 * Tree Traversal Methods
 *********************************/

func (doc *DocumentLink) GetChild(name string) (any, bool) {
	switch name {
	case "author":
		return &doc.Author, true
	default:
		return nil, false
	}
}
