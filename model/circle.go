package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Circle is a grouping of people that is created/defined by a single UserID.
type Circle struct {
	CircleID    primitive.ObjectID `bson:"_id"`         // Unique identifier assigned by the database
	UserID      primitive.ObjectID `bson:"userId"`      // UserID of owner of this Circle
	Name        string             `bson:"name"`        // Human-readable name for this circle.
	Color       string             `bson:"color"`       // Color of this Circle, used to color the circle icon
	Icon        string             `bson:"icon"`        // Icon of this Circle, used to display the circle icon
	Description string             `bson:"description"` // Human-readable description of this Circle
	ProductIDs  id.Slice           `bson:"productIds"`  // List of remote ProductIDs that can purchase membership in this Circle
	MemberCount int64              `bson:"memberCount"` // Number of members in this Circle
	IsVisible   bool               `bson:"isVisible"`   // TRUE if members of this Circle can see that they're in this Circle.
	IsFeatured  bool               `bson:"isFeatured"`  // TRUE if this Circle should be featured on the User's profile page.

	journal.Journal `json:"-" bson:",inline"`
}

// NewCircle returns a fully initialized, empty Circle
func NewCircle() Circle {
	return Circle{
		CircleID: primitive.NewObjectID(),
	}
}

// CircleFields returns the database fields required to populate a Circle
func CircleFields() []string {
	return []string{"_id", "name", "icon", "color", "description", "productIds", "memberCount", "isFeatured"}
}

// Fields returns the database fields required to populate a Circle
func (circle Circle) Fields() []string {
	return CircleFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the primary key of this Circle, as a string
func (circle *Circle) ID() string {
	return circle.CircleID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Circle.
// It is part of the AccessLister interface
func (circle *Circle) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Circle
// It is part of the AccessLister interface
func (circle *Circle) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (circle *Circle) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == circle.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (circle *Circle) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(circle.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (circle *Circle) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Data Accessors
 ******************************************/

// HasProducts returns TRUE if this Circle is unlocked by purchasing a Product
func (circle Circle) HasProducts() bool {
	return circle.ProductIDs.NotEmpty()
}

// ProductCount returns the number of Products that unlock this Circle
func (circle Circle) ProductCount() int {
	return circle.ProductIDs.Length()
}

// Privileges returns every ID that grants access to this Circle: the Circle itself, plus any Products
func (circle Circle) Privileges() id.Slice {

	result := id.Slice{circle.CircleID}

	if circle.HasProducts() {
		result.Append(circle.ProductIDs...)
	}

	return result
}

// LookupCode returns this Circle as a form.LookupCode, so it can be listed in a picker
func (circle Circle) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       circle.CircleID.Hex(),
		Label:       circle.Name,
		Description: circle.Description,
		Icon:        circle.Icon,
	}
}
