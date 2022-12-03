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

func NewPasswordReset(duration time.Duration) PasswordReset {

	return PasswordReset{
		AuthCode:   random.String(64),
		CreateDate: time.Now().Unix(),
		ExpireDate: time.Now().Add(duration).Unix(),
	}
}

func PasswordResetSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"authCode":   schema.String{},
			"createDate": schema.Integer{},
			"expireDate": schema.Integer{},
		},
	}
}

// IsValid returns TRUE if the password reset code is valid and has not expired.
func (reset PasswordReset) IsValid(code string) bool {
	return (code != "") && (reset.AuthCode == code) && !reset.IsExpired()
}

// IsExpired returns TRUE if the password reset has expired
func (reset PasswordReset) IsExpired() bool {
	return reset.ExpireDate < time.Now().Unix()
}
