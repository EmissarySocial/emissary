package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Annotation is a grouping of people that is created/defined by a single UserID.
type Annotation struct {
	AnnotationID primitive.ObjectID `json:"annotationId"    bson:"_id"` // Unique identifier assigned by the database
	UserID       primitive.ObjectID `json:"userId"      bson:"userId"`  // UserID of owner of this Annotation
	URL          string             `json:"url"         bson:"url"`     // URL of this Annotation.
	Name         string             `json:"name"        bson:"name"`    // Name of the document being annotated.
	Icon         string             `json:"icon"        bson:"icon"`    // Icon of the document being annotated.
	Content      string             `json:"content"     bson:"content"` // Content of this Annotation

	journal.Journal `json:"-" bson:",inline"`
}

func NewAnnotation() Annotation {
	return Annotation{
		AnnotationID: primitive.NewObjectID(),
	}
}

func AnnotationFields() []string {
	return []string{"_id", "url", "name", "icon", "content"}
}

func (annotation Annotation) Fields() []string {
	return AnnotationFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

func (annotation *Annotation) ID() string {
	return annotation.AnnotationID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Annotation.
// It is part of the AccessLister interface
func (annotation *Annotation) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Annotation
// It is part of the AccessLister interface
func (annotation *Annotation) IsAuthor(authorID primitive.ObjectID) bool {
	return !authorID.IsZero() && authorID == annotation.UserID
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (annotation *Annotation) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (annotation *Annotation) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(annotation.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (annotation *Annotation) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/
