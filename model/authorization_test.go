package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestAuthorization_Revalidatable confirms that Authorization satisfies the
// steranko.Revalidatable interface, so Steranko can read its revalidation time.
func TestAuthorization_Revalidatable(t *testing.T) {
	var auth any = NewAuthorization()
	_, ok := auth.(steranko.Revalidatable)
	require.True(t, ok)
}

// TestAuthorization_GetRevalidationTime_Unset confirms that a session which has
// not opted in (Revalidate == 0) reports no revalidation time, so Steranko
// leaves it alone (e.g. guest/Identity sessions).
func TestAuthorization_GetRevalidationTime_Unset(t *testing.T) {
	auth := NewAuthorization()

	at, ok := auth.GetRevalidationTime()
	require.False(t, ok, "an unset Revalidate must report no time")
	require.True(t, at.IsZero())
}

// TestAuthorization_GetRevalidationTime_Set confirms that a populated Revalidate
// is reported back as the matching time.
func TestAuthorization_GetRevalidationTime_Set(t *testing.T) {
	auth := NewAuthorization()
	now := time.Now()
	auth.Revalidate = now.Unix()

	at, ok := auth.GetRevalidationTime()
	require.True(t, ok)
	require.Equal(t, now.Unix(), at.Unix())
}

// TestAuthorization_SessionCarrier confirms that Authorization satisfies the
// steranko.SessionCarrier interface.
func TestAuthorization_SessionCarrier(t *testing.T) {
	var auth any = &Authorization{}
	_, ok := auth.(steranko.SessionCarrier)
	require.True(t, ok)
}

// TestAuthorization_CarryForwardSessionState confirms that the security-critical Masquerade
// flag is copied forward, while user-record-derived fields (DomainOwner) are not.
func TestAuthorization_CarryForwardSessionState(t *testing.T) {

	// The previous session: a masquerading owner marker.
	previous := NewAuthorization()
	previous.Masquerade = true
	previous.DomainOwner = true

	// The freshly-rebuilt session (from the user record) does NOT know it was a masquerade,
	// and rebuilds DomainOwner from the record (here, false).
	fresh := NewAuthorization()
	fresh.Masquerade = false
	fresh.DomainOwner = false

	fresh.CarryForwardSessionState(&previous)

	require.True(t, fresh.Masquerade, "Masquerade must be carried forward across the re-mint")
	require.False(t, fresh.DomainOwner, "DomainOwner must NOT be carried forward; it is re-derived from the record")
}

// TestAuthorization_CarryForwardSessionState_WrongType confirms that carrying forward from
// an unrelated claims type is a safe no-op (no panic, no change).
func TestAuthorization_CarryForwardSessionState_WrongType(t *testing.T) {

	fresh := NewAuthorization()
	fresh.Masquerade = false

	require.NotPanics(t, func() {
		fresh.CarryForwardSessionState(jwt.MapClaims{"M": true})
	})
	require.False(t, fresh.Masquerade, "an unrelated claims type must not alter Masquerade")
}

// TestAuthorization_RevalidateRoundTrip confirms that both the "sub" subject and
// the "R" revalidation timestamp survive a JSON marshal/unmarshal -- the exact
// path a token takes through signing and parsing -- and that the result reads as
// stale when the timestamp is old.
func TestAuthorization_RevalidateRoundTrip(t *testing.T) {
	userID := primitive.NewObjectID()

	original := NewAuthorization()
	original.UserID = userID
	original.Subject = userID.Hex()
	original.Revalidate = time.Now().Add(-30 * time.Minute).Unix()

	encoded, err := json.Marshal(original)
	require.Nil(t, err)

	var parsed Authorization
	require.Nil(t, json.Unmarshal(encoded, &parsed))

	// The subject (used by LoadBySubject) must survive.
	subject, err := parsed.GetSubject()
	require.Nil(t, err)
	require.Equal(t, userID.Hex(), subject)

	// The revalidation time must survive and read as stale.
	at, ok := parsed.GetRevalidationTime()
	require.True(t, ok, "Revalidate must survive the JSON round-trip")
	require.Greater(t, time.Since(at), 10*time.Minute, "the round-tripped session must read as stale")
}
