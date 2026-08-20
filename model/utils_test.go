package model

import (
	"testing"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestToToken verifies that arbitrary text is reduced to a safe, URL-friendly token
func TestToToken(t *testing.T) {

	do := func(input, expected string) {
		require.Equal(t, ToToken(input), expected)
	}

	do("!!!! Ignore leading chars", "ignore-leading-chars")   // Ignore leading special characters
	do("Ignore trailing chars !!!!", "ignore-trailing-chars") // Ignore trailing special characters
	do("Hello, World!", "hello-world")                        // Lowercase, and replace special characters with "-"
	do("Hägen Däs", "hägen-däs")                              // Allow diacritics
	do("Æthelflad", "æthelflad")                              // Æthenflad is a bad-ass.
	do("category:value", "category:value")                    // Intentionally allowing ":" because it's used for tag categories
}

// TestDefaultRolesToGroupIDs covers the shared AccessLister role-resolution that most models
// delegate to. Each magic role maps to its magic group; "myself"/"author" map to the owner only
// when the owner is set; unknown roles contribute nothing.
func TestDefaultRolesToGroupIDs(t *testing.T) {

	owner := primitive.NewObjectID()

	// Anonymous role -> the anonymous magic group.
	require.Equal(t, Permissions{MagicGroupIDAnonymous}, defaultRolesToGroupIDs(owner, MagicRoleAnonymous))

	// Authenticated role -> the authenticated magic group.
	require.Equal(t, Permissions{MagicGroupIDAuthenticated}, defaultRolesToGroupIDs(owner, MagicRoleAuthenticated))

	// Myself and Author both resolve to the owner's ID when the owner is set.
	require.Equal(t, Permissions{owner}, defaultRolesToGroupIDs(owner, MagicRoleMyself))
	require.Equal(t, Permissions{owner}, defaultRolesToGroupIDs(owner, MagicRoleAuthor))

	// Multiple roles accumulate, in order.
	require.Equal(t,
		Permissions{MagicGroupIDAnonymous, MagicGroupIDAuthenticated, owner},
		defaultRolesToGroupIDs(owner, MagicRoleAnonymous, MagicRoleAuthenticated, MagicRoleMyself),
	)

	// Unknown roles contribute nothing.
	require.Equal(t, Permissions{}, defaultRolesToGroupIDs(owner, "not-a-real-role"))

	// No roles -> empty result.
	require.Equal(t, Permissions{}, defaultRolesToGroupIDs(owner))
}

// TestDefaultRolesToGroupIDs_ZeroOwner confirms the guard: when the owner is the zero ObjectID,
// "myself"/"author" resolve to nothing (a zero owner must never be granted owner permissions).
func TestDefaultRolesToGroupIDs_ZeroOwner(t *testing.T) {

	var zero primitive.ObjectID

	require.Equal(t, Permissions{}, defaultRolesToGroupIDs(zero, MagicRoleMyself))
	require.Equal(t, Permissions{}, defaultRolesToGroupIDs(zero, MagicRoleAuthor))

	// Non-owner roles still resolve normally even with a zero owner.
	require.Equal(t, Permissions{MagicGroupIDAnonymous}, defaultRolesToGroupIDs(zero, MagicRoleAnonymous))
}

// TestFlatten covers the helper that collapses a map of id.Slice values into a single id.Slice.
func TestFlatten(t *testing.T) {

	a := primitive.NewObjectID()
	b := primitive.NewObjectID()
	c := primitive.NewObjectID()

	// Empty map -> empty slice.
	require.Equal(t, id.Slice{}, flatten(mapof.Object[id.Slice]{}))

	// Single key.
	require.Equal(t, id.Slice{a}, flatten(mapof.Object[id.Slice]{"x": {a}}))

	// Multiple keys collapse into one slice; verify contents regardless of map order.
	result := flatten(mapof.Object[id.Slice]{"x": {a, b}, "y": {c}})
	require.Len(t, result, 3)
	require.Contains(t, result, a)
	require.Contains(t, result, b)
	require.Contains(t, result, c)
}

// TestOneOf covers the generic membership helpers.
func TestOneOf(t *testing.T) {

	require.True(t, oneOf("b", "a", "b", "c"))
	require.False(t, oneOf("z", "a", "b", "c"))
	require.False(t, oneOf("a")) // no options -> never matches

	require.False(t, notOneOf("b", "a", "b", "c"))
	require.True(t, notOneOf("z", "a", "b", "c"))

	require.True(t, oneOf(2, 1, 2, 3)) // works for non-string comparables
}

// TestMust confirms the error-stripping helper returns the value and discards the error.
func TestMust(t *testing.T) {
	require.Equal(t, "value", must("value", nil))
	require.Equal(t, 42, must(42, errExample))
}

// errExample is a sentinel error used by the tests in this file
var errExample = primitiveError("example")

// primitiveError is a string-backed error, used to build sentinel errors in these tests
type primitiveError string

// Error implements the error interface, returning the underlying string
func (e primitiveError) Error() string { return string(e) }
