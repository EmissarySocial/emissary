package build

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
)

// PasswordReset is a Builder for the "choose a new password" pages, which are reached
// from a one-time code in a password reset email.
type PasswordReset struct {
	_user model.User

	Theme
}

// NewPasswordReset returns a fully initialized PasswordReset builder.  The User must
// already have been loaded by the caller.
func NewPasswordReset(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, user model.User) PasswordReset {

	// RULE: This carries its own User rather than reusing one of the user Builders.  The
	// visitor is NOT signed in -- the reset code itself is the credential -- so borrowing
	// a user Builder would imply an authorization this page deliberately does not have.
	return PasswordReset{
		_user: user,
		Theme: NewTheme(factory, session, request, response),
	}
}

// UserID returns the ID of the User whose password is being reset.
func (w PasswordReset) UserID() string {

	// Read the loaded User, not the URL: a reset link may address a User by username, and
	// the value posted back must name the record that was actually verified.
	return w._user.UserID.Hex()
}

// Username returns the username of the User whose password is being reset.
func (w PasswordReset) Username() string {
	return w._user.Username
}

// Code returns the one-time reset code from the request URL, so that the form can post it
// back alongside the new password.
func (w PasswordReset) Code() string {
	return w.QueryParam("code")
}
