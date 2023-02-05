package model

import "strings"

// ActivityStreamContainer is an enum that represents the location of an Activity.
// The enum is designed according to: https://www.sohamkamani.com/golang/enums/
type ActivityStreamContainer int

// ActivityStreamContainerUndefined represents an Activity that is stored in an undefined location
const ActivityStreamContainerUndefined ActivityStreamContainer = 0

// ActivityStreamContainerInbox represents an Activity that is stored in a User's Inbox
const ActivityStreamContainerInbox ActivityStreamContainer = 1

// ActivityStreamContainerOutbox represents an Activity that is stored in a User's Outbox
const ActivityStreamContainerOutbox ActivityStreamContainer = 2

// ActivityStreamContainerFollowers represents an Activity that is stored in a User's Followers Outbox
const ActivityStreamContainerFollowers ActivityStreamContainer = 3

// ActivityStreamContainerFollowing represents an Activity that is stored in a User's Following Outbox
const ActivityStreamContainerFollowing ActivityStreamContainer = 4

// ActivityStreamContainerLiked represents an Activity that is stored in a User's Liked Outbox
const ActivityStreamContainerLiked ActivityStreamContainer = 5

// ActivityStreamContainerKeys represents an Activity that is stored in a User's Key Outbox
const ActivityStreamContainerKeys ActivityStreamContainer = 6

// ParseActivityStreamContainer converts a string into an ActivityStreamContainer
func ParseActivityStreamContainer(value string) ActivityStreamContainer {

	switch strings.ToLower(value) {
	case "inbox":
		return ActivityStreamContainerInbox
	case "outbox":
		return ActivityStreamContainerOutbox
	case "followers":
		return ActivityStreamContainerFollowers
	case "following":
		return ActivityStreamContainerFollowing
	case "liked":
		return ActivityStreamContainerLiked
	case "keys":
		return ActivityStreamContainerKeys
	default:
		return ActivityStreamContainerUndefined
	}
}

// String implements the Stringer interface
func (activityPlace ActivityStreamContainer) String() string {

	switch activityPlace {
	case ActivityStreamContainerInbox:
		return "inbox"
	case ActivityStreamContainerOutbox:
		return "outbox"
	case ActivityStreamContainerFollowers:
		return "followers"
	case ActivityStreamContainerFollowing:
		return "following"
	case ActivityStreamContainerLiked:
		return "liked"
	case ActivityStreamContainerKeys:
		return "keys"
	default:
		return "undefined"
	}
}
