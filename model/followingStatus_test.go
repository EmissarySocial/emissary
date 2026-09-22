package model

import (
	"slices"
	"testing"

	"github.com/benpate/delta"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/stretchr/testify/require"
)

// everyFollowingStatus lists every status a Following record can hold.
func everyFollowingStatus() []string {
	return []string{
		FollowingStatusNew,
		FollowingStatusLoading,
		FollowingStatusImportPending,
		FollowingStatusSuccess,
		FollowingStatusFailure,
		FollowingStatusPaused,
		FollowingStatusGone,
		FollowingStatusBlocked,
	}
}

// TestFollowingStatusLabel pins the words a Following's owner reads. Until BUG-148 the list
// template branched on a status string ("ERROR") that does not exist, so a FAILURE fell through
// to the final branch and rendered GREEN.
func TestFollowingStatusLabel(t *testing.T) {

	require.Equal(t, "Failed Permanently", followingStatusLabel(FollowingStatusGone))
	require.Equal(t, "Paused", followingStatusLabel(FollowingStatusPaused))
	require.Equal(t, "Not Responding", followingStatusLabel(FollowingStatusFailure))
	require.Equal(t, "Connected", followingStatusLabel(FollowingStatusSuccess))
	require.Equal(t, "Blocked", followingStatusLabel(FollowingStatusBlocked))

	// RULE: Every status has a human label, and none of them leaks the raw constant
	for _, status := range everyFollowingStatus() {
		label := followingStatusLabel(status)
		require.NotEmpty(t, label, "status %s", status)
		require.NotEqual(t, status, label, "status %s still shows its raw value", status)
	}

	// An unrecognized status shows its raw value rather than an empty badge
	require.Equal(t, "WAT", followingStatusLabel("WAT"))
	require.Empty(t, followingStatusLabel(""))
}

// TestFollowingStatusClass pins the colour. A status that falls through to the wrong branch is
// how a broken follow came to render green, so the mapping is asserted for every status.
func TestFollowingStatusClass(t *testing.T) {

	// RULE: All three failure states are red. PAUSED is still a failure the user should see,
	// not a neutral resting state; BLOCKED is the user's own decision and is not.
	require.Equal(t, "red", followingStatusClass(FollowingStatusGone))
	require.Equal(t, "red", followingStatusClass(FollowingStatusPaused))
	require.Equal(t, "red", followingStatusClass(FollowingStatusFailure))
	require.Equal(t, "light-gray", followingStatusClass(FollowingStatusBlocked))

	// RULE: ONLY a success is green
	require.Equal(t, "green", followingStatusClass(FollowingStatusSuccess))

	for _, status := range everyFollowingStatus() {
		if status != FollowingStatusSuccess {
			require.NotEqual(t, "green", followingStatusClass(status), "status %s must not read as healthy", status)
		}
	}

	// Everything else is neutral, and nothing is ever blank -- "text-" alone is not a class
	require.Equal(t, "light-gray", followingStatusClass(FollowingStatusNew))
	require.Equal(t, "light-gray", followingStatusClass(""))

	for _, status := range append(everyFollowingStatus(), "", "WAT") {
		require.NotEmpty(t, followingStatusClass(status), "status %s", status)
	}
}

// TestFollowingStatusDescription pins the one-line reason shown beside a problem badge. It is
// static per status by design: the dynamic StatusMessage can be a full DNS or TLS error and
// belongs in the detail panel, not a compact row.
func TestFollowingStatusDescription(t *testing.T) {

	require.Equal(t, "Not responding for 30+ days", followingStatusDescription(FollowingStatusPaused))
	require.Equal(t, "This account was deleted", followingStatusDescription(FollowingStatusGone))
	require.Equal(t, "You blocked this account", followingStatusDescription(FollowingStatusBlocked))
	require.Equal(t, "Last check failed", followingStatusDescription(FollowingStatusFailure))

	// RULE: Every red badge has a reason beside it, and no healthy badge does
	for _, status := range everyFollowingStatus() {

		if followingStatusClass(status) == "red" {
			require.NotEmpty(t, followingStatusDescription(status), "problem status %s must explain itself", status)
		}
	}

	require.Empty(t, followingStatusDescription(FollowingStatusSuccess))
	require.Empty(t, followingStatusDescription(FollowingStatusNew))
	require.Empty(t, followingStatusDescription("WAT"))
}

// TestFollowingStatusAccessorsAgree confirms the list view and the detail view can never
// describe one record differently, since they read different types.
func TestFollowingStatusAccessorsAgree(t *testing.T) {

	for _, status := range everyFollowingStatus() {

		following := Following{Status: status}
		summary := FollowingSummary{Status: status}

		require.Equal(t, following.StatusLabel(), summary.StatusLabel(), "status %s", status)
		require.Equal(t, following.StatusClass(), summary.StatusClass(), "status %s", status)
		require.Equal(t, following.StatusDescription(), summary.StatusDescription(), "status %s", status)
	}
}

// serverOnlyFollowingStatuses lists the statuses that are written only by the server and never
// arrive through a form. The schema enum is the FORM INPUT contract, so these stay out of it on
// purpose -- the same convention FollowerStateBlocked follows (D6).
func serverOnlyFollowingStatuses() []string {
	return []string{FollowingStatusBlocked}
}

// TestFollowingStatusInSchema pins the enum in both directions. Every editable status must
// validate, or Following.Save refuses the record with "Must be one of the specified values".
// Every server-only status must NOT validate, or a form could write it (D6).
func TestFollowingStatusInSchema(t *testing.T) {

	followingSchema := schema.New(FollowingSchema())
	serverOnly := serverOnlyFollowingStatuses()

	validate := func(status string) error {
		following := NewFollowing()
		following.Status = status
		following.FolderID = delta.NewObjectID(primitive.NewObjectID())
		following.URL = "https://example.social/@someone"
		_, err := followingSchema.Validate(&following)
		return err
	}

	for _, status := range everyFollowingStatus() {

		if slices.Contains(serverOnly, status) {
			require.Error(t, validate(status), "server-only status %s must be absent from the schema enum (D6)", status)
			continue
		}

		require.NoError(t, validate(status), "status %s is missing from the schema enum", status)
	}
}
