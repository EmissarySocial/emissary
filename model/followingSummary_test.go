package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFollowingSummaryIcon_ProblemStatusesAlert pins the icon half of a problem badge. The list
// row colours the icon with StatusClass, so a red status that keeps its ordinary method icon
// renders as a red RSS logo, which reads as decoration rather than as a warning.
func TestFollowingSummaryIcon_ProblemStatusesAlert(t *testing.T) {

	// RULE: Every status the model calls "red" shows the SAME alert icon. The three problem
	// states differ in how long we keep trying, never in how loudly they say something broke.
	for _, status := range everyFollowingStatus() {

		if followingStatusClass(status) != "red" {
			continue
		}

		for _, method := range []string{FollowingMethodActivityPub, FollowingMethodPoll, ""} {
			summary := FollowingSummary{Status: status, Method: method}
			require.Equal(t, "alert-fill", summary.Icon(), "status %s / method %q", status, method)
		}
	}
}

// TestFollowingSummaryIcon_HealthyStatuses confirms a working follow still shows WHICH protocol
// it uses, which is the only place the list says so.
func TestFollowingSummaryIcon_HealthyStatuses(t *testing.T) {

	require.Equal(t, "activitypub-fill", FollowingSummary{Status: FollowingStatusSuccess, Method: FollowingMethodActivityPub}.Icon())
	require.Equal(t, "rss-fill", FollowingSummary{Status: FollowingStatusSuccess, Method: FollowingMethodPoll}.Icon())

	// LOADING is a spinner regardless of protocol, because there is not yet anything to report
	require.Equal(t, "loading", FollowingSummary{Status: FollowingStatusLoading, Method: FollowingMethodPoll}.Icon())

	// BLOCKED is the user's own decision, so it is not dressed up as a failure
	require.NotEqual(t, "alert-fill", FollowingSummary{Status: FollowingStatusBlocked, Method: FollowingMethodPoll}.Icon())
}

// TestFollowingSummaryIcon_NeverEmpty covers the gap a half-connected record falls into: Method
// is blank until Connect resolves it, and "{{icon \"\"}}" renders nothing at all.
func TestFollowingSummaryIcon_NeverEmpty(t *testing.T) {

	for _, status := range everyFollowingStatus() {
		for _, method := range []string{FollowingMethodActivityPub, FollowingMethodPoll, ""} {

			icon := FollowingSummary{Status: status, Method: method}.Icon()

			require.NotEmpty(t, icon, "status %s / method %q renders no icon at all", status, method)
			require.False(t, strings.HasPrefix(icon, "-"), "status %s / method %q builds a broken name %q", status, method, icon)
		}
	}
}
