package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AnnotationSchema returns a validating schema for Annotation objects.
func AnnotationSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"annotationId": schema.String{Format: "objectId"},
			"userId":       schema.String{Format: "objectId"},
			"url":          schema.String{Format: "url"},
			"name":         schema.String{MaxLength: 255},
			"icon":         schema.String{MaxLength: 255},
			"content":      schema.String{MaxLength: 10000},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetPointer implements the schema.PointerGetter interface, and
// allows read/write access to (most) fields of the Annotation object.
func (annotation *Annotation) GetPointer(name string) (any, bool) {

	switch name {

	case "url":
		return &annotation.URL, true

	case "name":
		return &annotation.Name, true

	case "icon":
		return &annotation.Icon, true

	case "content":
		return &annotation.Content, true

	}

	return "", false
}

// GetStringOK implements the schema.StringGetter interface, and
// returns string values for several fields of the Annotation object.
func (annotation *Annotation) GetStringOK(name string) (string, bool) {

	switch name {

	case "annotationId":
		return annotation.AnnotationID.Hex(), true

	case "userId":
		return annotation.UserID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetString implemments the schema.StringSetter interface, and
// allows setting string values for several fields of the Annotation object.
func (annotation *Annotation) SetString(name string, value string) bool {

	switch name {

	case "annotationId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			annotation.AnnotationID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			annotation.UserID = objectID
			return true
		}
	}

	return false
}
