package model

/******************************************
 * Following Status Display
 *
 * The display vocabulary for a Following's Status,
 * shared by model.Following and model.FollowingSummary
 * so that the list view and the detail view can never
 * describe the same record differently.
 ******************************************/

// followingStatusClass returns the CSS color suffix for a Following status, for use as "text-<class>"
func followingStatusClass(status string) string {

	switch status {

	case FollowingStatusSuccess:
		return "green"

	case FollowingStatusFailure, FollowingStatusPaused, FollowingStatusGone:
		return "red"
	}

	return "light-gray"
}

// followingStatusDescription returns one short sentence saying WHY a Following is in a problem
// status, for the compact list row
func followingStatusDescription(status string) string {

	// This is static per status by design.  The record's own StatusMessage carries the HTTP
	// code or transport reason, which is too long for a row and belongs in the detail panel.
	switch status {

	case FollowingStatusFailure:
		return "Last check failed"

	case FollowingStatusPaused:
		return "Not responding for 30+ days"

	case FollowingStatusGone:
		return "This account was deleted"

	case FollowingStatusBlocked:
		return "You blocked this account"
	}

	// A healthy or in-progress status needs no explanation
	return ""
}

// followingStatusLabel returns the human-readable description of a Following status
func followingStatusLabel(status string) string {

	switch status {

	case FollowingStatusNew:
		return "New"

	case FollowingStatusLoading:
		return "Connecting"

	case FollowingStatusImportPending:
		return "Imported"

	case FollowingStatusSuccess:
		return "Connected"

	case FollowingStatusFailure:
		return "Not Responding"

	case FollowingStatusPaused:
		return "Paused"

	case FollowingStatusGone:
		return "Failed Permanently"

	case FollowingStatusBlocked:
		return "Blocked"
	}

	// An unrecognized status shows its own raw value rather than an empty badge
	return status
}
