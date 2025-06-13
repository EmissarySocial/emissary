package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Circle is a grouping of people that is created/defined by a single UserID.
type Circle struct {
	CircleID    primitive.ObjectID `json:"circleId"    bson:"_id"`         // Unique identifier assigned by the database
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"`      // UserID of owner of this Circle
	Name        string             `json:"name"        bson:"name"`        // Human-readable name for this circle.
	Color       string             `json:"color"       bson:"color"`       // Color of this Circle, used to color the circle icon
	Icon        string             `json:"icon"        bson:"icon"`        // Icon of this Circle, used to display the circle icon
	Description string             `json:"description" bson:"description"` // Human-readable description of this Circle
	ProductIDs  sliceof.String     `json:"productIds"  bson:"productIds"`  // List of remote ProductIDs that can purchase membership in this Circle
	MemberCount int64              `json:"memberCount" bson:"memberCount"` // Number of members in this Circle
	IsFeatured  bool               `json:"isFeatured"  bson:"isFeatured"`  // TRUE if this Circle should be featured on the User's profile page.

	journal.Journal `json:"-" bson:",inline"`
}

func NewCircle() Circle {
	return Circle{
		CircleID: primitive.NewObjectID(),
	}
}

func CircleFields() []string {
	return []string{"_id", "name", "icon", "color", "description", "productIds", "memberCount", "isFeatured"}
}

func (userSummary Circle) Fields() []string {
	return CircleFields()
}

/******************************************
 * data.Object Interface
 ******************************************/

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
	return userID == circle.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (circle *Circle) RolesToGroupIDs(roleIDs ...string) id.Slice {
	return nil
}

// RolesToPrivileges returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (circle *Circle) RolesToPrivileges(roleIDs ...string) sliceof.String {
	return sliceof.NewString()
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (circle Circle) HasProducts() bool {
	return circle.ProductIDs.NotEmpty()
}

func (circle Circle) ProductCount() int {
	return circle.ProductIDs.Length()
}

func (circle Circle) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       "CIR:" + circle.CircleID.Hex(),
		Label:       circle.Name,
		Description: circle.Description,
		Icon:        circle.Icon,
		Group:       "Circles",
	}
}
