package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Password chain mechanics (primary selection, rehash flagging, truncation, fuzz) are
// tested in the steranko package, where the PasswordService lives.  These tests pin
// the POLICY that Factory.Steranko configures — exercised through the real factory
// wiring, so a config change here cannot slip through unnoticed.  Hashing never
// touches the database, so a zero Factory and a nil session are safe.

// TestSteranko_SetPassword_StoresBCrypt12 is the regression test for the plaintext
// password vulnerability (CWE-256): registration, password reset, and Mastodon signup
// once stored raw passwords because they bypassed the hasher.  All password writes now
// route through the Steranko instance built here, so this pins that a User can only
// ever receive a bcrypt hash — at cost 12, the deliberate balance between
// offline-cracking resistance and signin latency / failed-signin CPU cost.
func TestSteranko_SetPassword_StoresBCrypt12(t *testing.T) {

	factory := Factory{}
	steranko := factory.Steranko(nil)

	user := model.NewUser()
	require.Nil(t, steranko.SetPassword(&user, "TestPass123!"))

	require.NotEqual(t, "TestPass123!", user.Password)
	require.Nil(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("TestPass123!")))

	cost, err := bcrypt.Cost([]byte(user.Password))
	require.Nil(t, err, "stored password must be a bcrypt hash: %q", user.Password)
	require.Equal(t, 12, cost)
}

// TestSteranko_PlaintextFallback pins that legacy plaintext-stored passwords still
// verify — flagged for re-hashing — so pre-hashing accounts can sign in.  When the
// plaintext-password migration ships, the fallback is removed from Factory.Steranko
// and this test must be DELETED deliberately.
func TestSteranko_PlaintextFallback(t *testing.T) {

	factory := Factory{}
	steranko := factory.Steranko(nil)

	ok, rehash := steranko.ComparePassword("legacy-password", "legacy-password")
	require.True(t, ok)
	require.True(t, rehash, "plaintext matches must be flagged for upgrade")
}
