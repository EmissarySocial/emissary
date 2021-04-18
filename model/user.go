package model

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/ghost/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person or machine account that can own pages and sections.
type User struct {
	UserID      primitive.ObjectID   `json:"userId"      bson:"_id"`         // Unique identifier for this user.
	GroupIDs    []primitive.ObjectID `json:"groupIds"    bson:"groupIds"`    // Slice of IDs for the groups that this user belongs to.
	DisplayName string               `json:"displayName" bson:"displayName"` // Name to be displayed for this user
	Username    string               `json:"username"    bson:"username"`    // This is the primary public identifier for the user.
	Password    string               `json:"password"    bson:"password"`    // This password should be encrypted with BCrypt.
	IsOwner     bool                 `json:"isOwner"     bson:"isOwner"`     // If TRUE, then this user is a website owner with FULL privileges.
	ImageURL    string               `json:"imageUrl"    bson:"imageUrl"`    // Avatar image of this user.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}

//////////////////////////
// Steranko Interfaces

// GetUsername returns the username for this User.  It helps implement the "steranko.User" interface.
func (user *User) GetUsername() string {
	return user.Username
}

// GetPassword returns the (encrypted) passsword for this User.  It helps implement the "steranko.User" interface.
func (user *User) GetPassword() string {
	return user.Password
}

// SetUsername updates the username for this User.  It helps implement the "steranko.User" interface.
func (user *User) SetUsername(username string) {
	user.Username = username
}

// SetPassword updates the password for this User.  It helps implement the "steranko.User" interface.
func (user *User) SetPassword(password string) {
	user.Password = password
}

// Claims returns all access privileges given to this user
func (user *User) Claims() map[string]interface{} {

	result := map[string]interface{}{
		"sub": user.UserID.Hex(),                        // Subject of the JWT (UserID)
		"grp": datatype.ConvertObjectIDs(user.GroupIDs), // GroupIDs that this User belongs to
		"exp": time.Now().Add(time.Hour).Unix(),         // Expires one hour from now.  (To be refreshed by Steranko)
	}

	if user.IsOwner {
		result["owner"] = "true"
	}

	return result
}
