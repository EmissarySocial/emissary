package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BookmarkSchema returns the rosetta schema that describes a Bookmark
func BookmarkSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"bookmarkId": schema.String{Format: "objectId"},
			"userId":     schema.String{Format: "objectId"},
			"url":        schema.String{Format: "url", Required: true, MaxLength: 2048},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (bookmark *Bookmark) GetPointer(name string) (any, bool) {
	switch name {

	case "url":
		return &bookmark.URL, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (bookmark *Bookmark) GetStringOK(name string) (string, bool) {
	switch name {

	case "bookmarkId":
		return bookmark.BookmarkID.Hex(), true

	case "userId":
		return bookmark.UserID.Hex(), true
	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (bookmark *Bookmark) SetString(name string, value string) bool {
	switch name {

	case "bookmarkId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			bookmark.BookmarkID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			bookmark.UserID = objectID
			return true
		}
	}

	return false
}
