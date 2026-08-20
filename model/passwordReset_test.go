package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewPasswordReset verifies that a new reset code carries full entropy and is immediately usable
func TestNewPasswordReset(t *testing.T) {

	reset := NewPasswordReset(PasswordResetDurationReset)

	// A new reset code is 64 characters of entropy and immediately usable
	require.Len(t, reset.AuthCode, 64)
	require.True(t, reset.IsActive())
	require.False(t, reset.IsExpired())

	// CreateDate is "now" and ExpireDate honors the requested duration
	now := time.Now().Unix()
	require.InDelta(t, now, reset.CreateDate, 5)
	require.InDelta(t, now+int64(PasswordResetDurationReset.Seconds()), reset.ExpireDate, 5)
}

// TestNewPasswordReset_WelcomeDuration verifies that welcome and invite codes get the longer expiration window
func TestNewPasswordReset_WelcomeDuration(t *testing.T) {

	reset := NewPasswordReset(PasswordResetDurationWelcome)

	// Welcome/invite codes get the longer expiration window
	now := time.Now().Unix()
	require.InDelta(t, now+int64(PasswordResetDurationWelcome.Seconds()), reset.ExpireDate, 5)
}

// TestPasswordReset_RefreshExpireDate verifies that refreshing an expired code makes it usable again
func TestPasswordReset_RefreshExpireDate(t *testing.T) {

	reset := NewPasswordReset(PasswordResetDurationReset)
	reset.ExpireDate = time.Now().Add(-1 * time.Hour).Unix()
	require.True(t, reset.IsExpired())

	// Refreshing an expired code makes it usable for the new duration
	reset.RefreshExpireDate(PasswordResetDurationReset)

	now := time.Now().Unix()
	require.False(t, reset.IsExpired())
	require.InDelta(t, now+int64(PasswordResetDurationReset.Seconds()), reset.ExpireDate, 5)
}

// TestPasswordReset_IsValid verifies that only the matching code is accepted, and only before it expires
func TestPasswordReset_IsValid(t *testing.T) {

	reset := NewPasswordReset(PasswordResetDurationReset)

	// The matching code is valid until the reset expires
	require.True(t, reset.IsValid(reset.AuthCode))

	// Wrong and empty codes are always rejected
	require.False(t, reset.IsValid("WRONG-CODE"))
	require.False(t, reset.IsValid(""))

	// Expired codes are rejected even when they match
	expired := NewPasswordReset(PasswordResetDurationReset)
	expired.ExpireDate = time.Now().Add(-1 * time.Minute).Unix()
	require.False(t, expired.IsValid(expired.AuthCode))
}

// TestPasswordReset_IsValid_Consumed verifies that a consumed reset rejects every code, including an empty one
func TestPasswordReset_IsValid_Consumed(t *testing.T) {

	// A consumed (zeroed) reset rejects every code, including the empty string.
	// This guards the single-use rule: "" must never match an empty AuthCode.
	consumed := PasswordReset{}
	require.False(t, consumed.IsValid(""))
	require.False(t, consumed.IsValid("ANY-CODE"))
}

// TestPasswordReset_IsActive verifies which reset states count as active
func TestPasswordReset_IsActive(t *testing.T) {

	// Zero value (never issued, or consumed) is not active
	empty := PasswordReset{}
	require.False(t, empty.IsActive())
	require.True(t, empty.NotActive())

	// Freshly issued codes are active
	active := NewPasswordReset(PasswordResetDurationReset)
	require.True(t, active.IsActive())
	require.False(t, active.NotActive())

	// Expired codes are not active
	expired := NewPasswordReset(PasswordResetDurationReset)
	expired.ExpireDate = time.Now().Add(-1 * time.Minute).Unix()
	require.False(t, expired.IsActive())
	require.True(t, expired.NotActive())
}
