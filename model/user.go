package model

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/golang-jwt/jwt/v4"
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
	AvatarURL   string               `json:"avatarUrl"    bson:"avatarUrl"`  // Avatar image of this user.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (user *User) ID() string {
	return user.UserID.Hex()
}

/******************************
 Steranko Interfaces
*******************************/

// GetUsername returns the username for this User.  A part of the "steranko.User" interface.
func (user *User) GetUsername() string {
	return user.Username
}

// GetPassword returns the (encrypted) passsword for this User.  A part of the "steranko.User" interface.
func (user *User) GetPassword() string {
	return user.Password
}

// SetUsername updates the username for this User.  A part of the "steranko.User" interface.
func (user *User) SetUsername(username string) {
	user.Username = username
}

// SetPassword updates the password for this User.  A part of the "steranko.User" interface.
func (user *User) SetPassword(password string) {
	user.Password = password
}

// Claims returns all access privileges given to this user.  A part of the "steranko.User" interface.
func (user *User) Claims() jwt.Claims {

	result := Authorization{
		UserID:      user.UserID,
		GroupIDs:    user.GroupIDs,
		DomainOwner: user.IsOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                   // Current create date.  (Used by Steranko to refresh tokens)
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(10, 0, 0)), // Expires ten years from now (but re-validated sooner by Steranko)
		},
	}

	return result
}
