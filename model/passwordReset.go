package model

import (
	"time"

	"github.com/benpate/rosetta/schema"
	"github.com/labstack/gommon/random"
)

// PasswordReset represents a single password reset request.
// Only one password reset request is allowed per user.
type PasswordReset struct {
	AuthCode   string
	CreateDate int64 `json:"createDate"`
	ExpireDate int64 `json:"expireDate"`
}

// NewPasswordReset returns a fully initialized PasswordReset object.
func NewPasswordReset() PasswordReset {

	result := PasswordReset{
		AuthCode: random.String(64),
	}

	result.RefreshExpireDate()

	return result
}

// PasswordResetSchema returns the data schema for PasswordReset objects
func PasswordResetSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"authCode":   schema.String{},
			"createDate": schema.Integer{},
			"expireDate": schema.Integer{},
		},
	}
}

// RefreshExpireDate extends the expiration date of the password reset code by 24 hours.
func (reset *PasswordReset) RefreshExpireDate() {
	reset.CreateDate = time.Now().Unix()
	reset.ExpireDate = time.Now().Add(time.Hour * 24).Unix()
}

// IsActive returns TRUE if this code exists and has not expired (i.e. people can still use it to reset their password)
func (reset PasswordReset) IsActive() bool {

	if reset.AuthCode == "" {
		return false
	}

	if reset.IsExpired() {
		return false
	}

	return true
}

// NotActive returns TRUE if this code does not exist, or has expired (i.e. people cannot use it to reset their password)
func (reset PasswordReset) NotActive() bool {
	return !reset.IsActive()
}

// IsValid returns TRUE if the password reset code is valid and has not expired.
func (reset PasswordReset) IsValid(code string) bool {
	return (code != "") && (reset.AuthCode == code) && !reset.IsExpired()
}

// IsExpired returns TRUE if the password reset has expired
func (reset PasswordReset) IsExpired() bool {
	return reset.ExpireDate < time.Now().Unix()
}
