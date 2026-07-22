package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestRegistration_SetUserData_PasswordIsHashed is the regression test for the
// plaintext password vulnerability (CWE-256): this exact code path once copied
// txn.Password straight into user.Password, so every self-registered user's
// password was stored verbatim in the database.
func TestRegistration_SetUserData_PasswordIsHashed(t *testing.T) {

	service := Registration{}
	steranko := (&Factory{}).Steranko(nil)
	domain := model.NewDomain()
	user := model.NewUser()
	require.True(t, user.IsNew())

	txn := model.RegistrationTxn{Password: "TestPass123!"}
	require.Nil(t, service.setUserData(nil, nil, steranko, &domain, &user, txn, []string{"password"}))

	// The stored value is a bcrypt hash of the submitted password, never the plaintext
	require.NotEqual(t, "TestPass123!", user.Password)
	_, err := bcrypt.Cost([]byte(user.Password))
	require.Nil(t, err, "stored password must be a bcrypt hash: %q", user.Password)
	require.Nil(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("TestPass123!")))
}

// TestRegistration_SetUserData_PasswordIgnoredForExistingUser pins that the
// registration form can only set a password on a NEW user — an update to an
// existing user must not overwrite their password.
func TestRegistration_SetUserData_PasswordIgnoredForExistingUser(t *testing.T) {

	service := Registration{}
	steranko := (&Factory{}).Steranko(nil)
	domain := model.NewDomain()

	user := model.NewUser()
	user.CreateDate = 1234567890 // saved => not new
	user.Password = "$2a$12$existing-hash-value"
	require.False(t, user.IsNew())

	txn := model.RegistrationTxn{Password: "NewPass456!"}
	require.Nil(t, service.setUserData(nil, nil, steranko, &domain, &user, txn, []string{"password"}))

	require.Equal(t, "$2a$12$existing-hash-value", user.Password)
}

// TestRegistration_SetUserData_EmptyPasswordIgnored pins that an empty password in
// the transaction leaves the User untouched instead of hashing an empty string.
func TestRegistration_SetUserData_EmptyPasswordIgnored(t *testing.T) {

	service := Registration{}
	steranko := (&Factory{}).Steranko(nil)
	domain := model.NewDomain()
	user := model.NewUser()

	txn := model.RegistrationTxn{Password: ""}
	require.Nil(t, service.setUserData(nil, nil, steranko, &domain, &user, txn, []string{"password"}))

	require.Empty(t, user.Password)
}
