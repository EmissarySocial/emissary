package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Privilege represents a single person who is a member of a circle
type Privilege struct {
	PrivilegeID       primitive.ObjectID `bson:"_id"`                        // Unique ID of this Privilege record
	IdentityID        primitive.ObjectID `bson:"identityId"`                 // Unique ID of the identity that is a member of the Circle
	UserID            primitive.ObjectID `bson:"userId"`                     // Unique ID of the User who owns the Circle or MerchantAccount
	CircleID          primitive.ObjectID `bson:"circleId,omitzero"`          // Unique ID of the Circle that this membership is associated with (if any)
	MerchantAccountID primitive.ObjectID `bson:"merchantAccountId,omitzero"` // Unique ID of the Merchant Account that this Privilege is associated with (if any)
	ProductID         primitive.ObjectID `bson:"productId,omitzero"`         // Unique ID of the Product/Plan that this Privilege is associated with (if any)
	Name              string             `bson:"name"`                       // Human-readable name of the privilege (e.g. "Monthly Subscription", "Annual Subscription", etc.)
	PriceDescription  string             `bson:"priceDescription,omitzero"`  // Description of the price for this privilege (e.g. "Monthly Subscription", "Annual Subscription", etc.)
	RecurringType     string             `bson:"recurringType,omitzero"`     // Type of recurring payment (e.g. "ONETIME", "WEEK", "MONTH", "YEAR")
	RemotePersonID    string             `bson:"remotePersonId,omitzero"`    // ID generated by the merchant account for the user/member
	RemoteProductID   string             `bson:"remoteProductId,omitzero"`   // ID generated by the merchant account for the product/plan
	RemotePurchaseID  string             `bson:"remotePurchaseId,omitzero"`  // ID generated by the merchant account for the purchase/purchase
	IdentifierType    string             `bson:"identifierType"`             // Type of Identifier that this Privilege is bound to (EmailAddress or ActivityPubActor)
	IdentifierValue   string             `bson:"identifierValue"`            // Value of the Identifier that this Privilege is bound to
	IsVisible         bool               `bson:"isVisible"`                  // TRUE if this Privilege is visible to the user/member

	// Embed journal to track changes
	journal.Journal `bson:",inline"`
}

func NewPrivilege() Privilege {
	return Privilege{
		PrivilegeID: primitive.NewObjectID(),
	}
}

func (privilege Privilege) ID() string {
	return privilege.PrivilegeID.Hex()
}

func (privilege Privilege) Fields() []string {
	return []string{
		"_id",
		"identityId",
		"userId",
		"circleId",
		"merchantAccountId",
		"productId",
		"name",
		"priceDescription",
		"recurringType",
		"identifierType",
		"identifierValue",
		"remotePersonId",
		"remoteProductId",
		"remotePurchaseId",
		"isVisible",
		"createDate",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Circle.
// It is part of the AccessLister interface
func (privilege *Privilege) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Privilege
// It is part of the AccessLister interface
func (privilege *Privilege) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (privilege *Privilege) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == privilege.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (privilege *Privilege) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(privilege.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (privilege *Privilege) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Getter Methods
 ******************************************/

func (privilege Privilege) IsPurchase() bool {
	return privilege.RemotePurchaseID != ""
}

func (privilege Privilege) IsRecurring() bool {
	switch privilege.RecurringType {

	case PrivilegeRecurringTypeDay,
		PrivilegeRecurringTypeWeek,
		PrivilegeRecurringTypeMonth,
		PrivilegeRecurringTypeYear:
		return true
	}

	return false
}

func (privilege Privilege) IsCircle() bool {
	return !privilege.CircleID.IsZero()
}

func (privilege Privilege) CompoundIDs() []primitive.ObjectID {
	result := make([]primitive.ObjectID, 0, 2)

	if !privilege.CircleID.IsZero() {
		result = append(result, privilege.CircleID)
	}

	if !privilege.ProductID.IsZero() {
		result = append(result, privilege.ProductID)
	}

	return result
}

/******************************************
 * Other Setter Methods
 ******************************************/

func (privilege *Privilege) SetCircleInfo(circle *Circle) bool {

	changed := false

	if privilege.UserID != circle.UserID {
		return false
	}

	if privilege.CircleID != circle.CircleID {
		privilege.CircleID = circle.CircleID
		changed = true
	}

	if privilege.Name != circle.Name {
		privilege.Name = circle.Name
		changed = true
	}

	if privilege.IsVisible != circle.IsVisible {
		privilege.IsVisible = circle.IsVisible
		changed = true
	}

	return changed
}
