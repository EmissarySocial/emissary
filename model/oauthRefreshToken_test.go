package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newGrant builds a grant whose refresh state is at the given generation, with
// known current and prior secrets, rotated at `rotatedAt`.
func newGrant(generation int, currentSecret string, priorSecret string, rotatedAt time.Time) *OAuthUserToken {
	grant := NewOAuthUserToken()
	grant.Generation = generation
	grant.RefreshHash = hashRefreshSecret(currentSecret)
	if priorSecret != "" {
		grant.RefreshPrevHash = hashRefreshSecret(priorSecret)
	}
	grant.RotatedAt = rotatedAt.Unix()
	return &grant
}

// TestRefreshToken_BuildParse_RoundTrip verifies that a built refresh token parses back into its parts
func TestRefreshToken_BuildParse_RoundTrip(t *testing.T) {

	id := primitive.NewObjectID()

	for _, generation := range []int{1, 2, 47, 1000} {
		token := BuildRefreshToken(id, generation, "abc-DEF_123")
		gotID, gotGen, gotSecret, err := ParseRefreshToken(token)

		require.Nil(t, err)
		require.Equal(t, id, gotID)
		require.Equal(t, generation, gotGen)
		require.Equal(t, "abc-DEF_123", gotSecret)
	}
}

// TestParseRefreshToken_Malformed verifies that every malformed token shape is rejected
func TestParseRefreshToken_Malformed(t *testing.T) {

	id := primitive.NewObjectID().Hex()

	bad := func(name, token string) {
		t.Run(name, func(t *testing.T) {
			_, _, _, err := ParseRefreshToken(token)
			require.NotNil(t, err, "token %q should be rejected", token)
		})
	}

	bad("empty", "")
	bad("one part", id)
	bad("two parts", id+".1")
	bad("bad grant id", "not-an-objectid.1.secret")
	bad("non-numeric generation", id+".x.secret")
	bad("zero generation", id+".0.secret")
	bad("negative generation", id+".-1.secret")
	bad("empty secret", id+".1.")

	// A secret containing extra dots is preserved (SplitN keeps the tail intact).
	t.Run("secret is never split", func(t *testing.T) {
		_, _, secret, err := ParseRefreshToken(id + ".1.a.b.c")
		require.Nil(t, err)
		require.Equal(t, "a.b.c", secret)
	})
}

// TestNewRefreshSecret verifies that generated secrets are the expected length and do not repeat
func TestNewRefreshSecret(t *testing.T) {
	first, err := NewRefreshSecret()
	require.Nil(t, err)
	require.NotEmpty(t, first)

	second, err := NewRefreshSecret()
	require.Nil(t, err)
	require.NotEqual(t, first, second, "secrets must be unique")
}

// TestRefreshHashMatches verifies that a secret matches its own hash, and nothing else
func TestRefreshHashMatches(t *testing.T) {
	hash := hashRefreshSecret("the-secret")

	require.True(t, refreshHashMatches(hash, "the-secret"))
	require.False(t, refreshHashMatches(hash, "wrong-secret"))
	require.False(t, refreshHashMatches("", "the-secret"), "empty stored hash never matches")
	require.False(t, refreshHashMatches(hash, ""), "empty secret never matches a real hash")
}

// TestClassifyRefresh verifies how a presented token is classified against the grant's current and previous generation
func TestClassifyRefresh(t *testing.T) {

	base := time.Unix(1_700_000_000, 0)
	grace := OAuthRefreshGracePeriod

	// A grant at generation 5, rotated at `base`, with current secret "cur5" and
	// prior secret "prev4".
	grant := newGrant(5, "cur5", "prev4", base)

	t.Run("current generation + current secret => Current", func(t *testing.T) {
		require.Equal(t, RefreshMatchCurrent, grant.ClassifyRefresh(5, "cur5", base, grace))
	})

	t.Run("prior generation + prior secret within grace => Grace", func(t *testing.T) {
		require.Equal(t, RefreshMatchGrace, grant.ClassifyRefresh(4, "prev4", base.Add(1*time.Minute), grace))
	})

	t.Run("prior generation + prior secret at grace boundary => Grace", func(t *testing.T) {
		require.Equal(t, RefreshMatchGrace, grant.ClassifyRefresh(4, "prev4", base.Add(grace), grace))
	})

	t.Run("prior generation + prior secret past grace => Reuse", func(t *testing.T) {
		require.Equal(t, RefreshMatchReuse, grant.ClassifyRefresh(4, "prev4", base.Add(grace+time.Second), grace))
	})

	// DoS guard: a wrong secret must never trigger the reuse alarm, even at a prior
	// or older generation — otherwise a guessed grant ID could force a revocation.
	t.Run("prior generation + WRONG secret => None (no alarm)", func(t *testing.T) {
		require.Equal(t, RefreshMatchNone, grant.ClassifyRefresh(4, "garbage", base.Add(time.Hour), grace))
	})

	t.Run("older generation + any secret => None (unconfirmable)", func(t *testing.T) {
		require.Equal(t, RefreshMatchNone, grant.ClassifyRefresh(3, "prev4", base.Add(time.Hour), grace))
		require.Equal(t, RefreshMatchNone, grant.ClassifyRefresh(1, "cur5", base.Add(time.Hour), grace))
	})

	t.Run("current generation + wrong secret => None", func(t *testing.T) {
		require.Equal(t, RefreshMatchNone, grant.ClassifyRefresh(5, "garbage", base, grace))
	})

	t.Run("future generation => None", func(t *testing.T) {
		require.Equal(t, RefreshMatchNone, grant.ClassifyRefresh(6, "cur5", base, grace))
	})
}

// TestInitRefresh verifies the state a grant is left in when its refresh chain is first created
func TestInitRefresh(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	grant := NewOAuthUserToken()
	grant.InitRefresh("secret-1", now)

	require.Equal(t, 1, grant.Generation)
	require.Equal(t, hashRefreshSecret("secret-1"), grant.RefreshHash)
	require.Empty(t, grant.RefreshPrevHash)
	require.Equal(t, now.Unix(), grant.RotatedAt)
}

// TestRotateRefresh verifies that rotating a grant advances the generation and retains the previous secret
func TestRotateRefresh(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	grant := NewOAuthUserToken()
	grant.InitRefresh("secret-1", start)

	later := start.Add(30 * time.Minute)
	grant.RotateRefresh("secret-2", later)

	require.Equal(t, 2, grant.Generation)
	require.Equal(t, hashRefreshSecret("secret-2"), grant.RefreshHash)
	require.Equal(t, hashRefreshSecret("secret-1"), grant.RefreshPrevHash, "prior secret is preserved for the grace window")
	require.Equal(t, later.Unix(), grant.RotatedAt)
}

// TestRefreshLifecycle exercises the full init -> use -> rotate -> reuse arc.
func TestRefreshLifecycle(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	grace := OAuthRefreshGracePeriod

	grant := NewOAuthUserToken()
	grant.InitRefresh("s1", base)

	// The freshly-issued secret is current.
	require.Equal(t, RefreshMatchCurrent, grant.ClassifyRefresh(1, "s1", base, grace))

	// Rotate to generation 2. s1 becomes the prior secret.
	rotateTime := base.Add(time.Minute)
	grant.RotateRefresh("s2", rotateTime)

	require.Equal(t, RefreshMatchCurrent, grant.ClassifyRefresh(2, "s2", rotateTime, grace))

	// Presenting the now-superseded s1 within the grace window is tolerated.
	require.Equal(t, RefreshMatchGrace, grant.ClassifyRefresh(1, "s1", rotateTime.Add(time.Minute), grace))

	// Presenting s1 past the grace window is a reuse alarm.
	require.Equal(t, RefreshMatchReuse, grant.ClassifyRefresh(1, "s1", rotateTime.Add(grace+time.Minute), grace))
}
