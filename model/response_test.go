package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestResponse verifies that every Response property round-trips through the schema
func TestResponse(t *testing.T) {

	s := schema.New(ResponseSchema())
	response := NewResponse()

	tests := []tableTestItem{
		{"responseId", "000000000000000000000001", nil},
		{"userId", "000000000000000000000001", nil},
		{"type", vocab.ActivityTypeAnnounce, nil},
		{"actor", "http://actor.com", nil},
		{"object", "https://example/object", nil},
		{"content", "😀", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}

/******************************************
 * ConflictingResponseTypes
 ******************************************/

// A Like displaces a previous Like (a repeat) and a previous Dislike (a contradiction).
func TestConflictingResponseTypes_Like(t *testing.T) {
	require.ElementsMatch(t,
		[]string{vocab.ActivityTypeLike, vocab.ActivityTypeDislike},
		ConflictingResponseTypes(vocab.ActivityTypeLike),
	)
}

// A Dislike displaces exactly the same set, so the rule is symmetric.
func TestConflictingResponseTypes_Dislike(t *testing.T) {
	require.ElementsMatch(t,
		[]string{vocab.ActivityTypeLike, vocab.ActivityTypeDislike},
		ConflictingResponseTypes(vocab.ActivityTypeDislike),
	)
}

// An Announce is independent: sharing a post you also liked is not a contradiction.
func TestConflictingResponseTypes_Announce(t *testing.T) {
	require.Equal(t,
		[]string{vocab.ActivityTypeAnnounce},
		ConflictingResponseTypes(vocab.ActivityTypeAnnounce),
	)
}

// An unrecognized type conflicts with itself alone, so it still cannot duplicate.
func TestConflictingResponseTypes_Unknown(t *testing.T) {
	require.Equal(t, []string{"Bookmark"}, ConflictingResponseTypes("Bookmark"))
	require.Equal(t, []string{""}, ConflictingResponseTypes(""))
}

// Like and Dislike must return identical slices, in identical ORDER.  queries/sync joins this
// result into a map key to bucket the reactions that displace each other, so a difference in
// order would split the pair into two buckets and let a contradiction survive the cleanup.
func TestConflictingResponseTypes_LikeAndDislikeAreIdentical(t *testing.T) {
	require.Equal(t,
		ConflictingResponseTypes(vocab.ActivityTypeLike),
		ConflictingResponseTypes(vocab.ActivityTypeDislike),
	)
}

// The result always includes the type itself, so re-reacting replaces instead of duplicating.
func TestConflictingResponseTypes_AlwaysIncludesItself(t *testing.T) {

	for _, responseType := range []string{
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike,
		vocab.ActivityTypeAnnounce,
		"Bookmark",
		"",
	} {
		require.Contains(t, ConflictingResponseTypes(responseType), responseType)
	}
}
