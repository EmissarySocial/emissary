package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Annotation is one User's private record of an ActivityPub object, with an optional note.
// It is never federated, and every User has at most one per URL.
type Annotation struct {
	AnnotationID primitive.ObjectID `json:"annotationId" bson:"_id"`     // Unique identifier assigned by the database
	UserID       primitive.ObjectID `json:"userId"       bson:"userId"`  // UserID of owner of this Annotation
	URL          string             `json:"url"          bson:"url"`     // URL of this Annotation.
	Type         string             `json:"type"         bson:"type"`    // ACTOR or DOCUMENT, stamped when the Annotation is created
	Name         string             `json:"name"         bson:"name"`    // Name of the document being annotated.
	Icon         string             `json:"icon"         bson:"icon"`    // Icon of the document being annotated.
	Content      string             `json:"content"      bson:"content"` // Content of this Annotation

	journal.Journal `json:"-" bson:",inline"`
}

// NewAnnotation returns a fully initialized, empty Annotation
func NewAnnotation() Annotation {
	return Annotation{
		AnnotationID: primitive.NewObjectID(),
	}
}

// AnnotationFields returns the database fields required to populate a Annotation
func AnnotationFields() []string {
	return []string{"_id", "userId", "url", "type", "name", "icon", "content", "createDate"}
}

// Fields returns the database fields required to populate a Annotation
func (annotation Annotation) Fields() []string {
	return AnnotationFields()
}

// IsActor returns TRUE if this Annotation points at an ActivityPub actor.
// The annotations list uses it to open a profile instead of a post.
func (annotation Annotation) IsActor() bool {
	return annotation.Type == AnnotationTypeActor
}

// NotActor returns TRUE if this Annotation does not point at an ActivityPub actor.
func (annotation Annotation) NotActor() bool {
	return !annotation.IsActor()
}

// IsDocument returns TRUE if this Annotation points at a document (anything that is not an actor).
func (annotation Annotation) IsDocument() bool {
	return annotation.Type == AnnotationTypeDocument
}

// NotDocument returns TRUE if this Annotation does not point at a document.
func (annotation Annotation) NotDocument() bool {
	return !annotation.IsDocument()
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this Annotation, as a string
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
