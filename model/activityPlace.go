package model

import "strings"

// ActivityPlace is an enum that represents the location of an Activity.
// The enum is designed according to: https://www.sohamkamani.com/golang/enums/
type ActivityPlace int

// ActivityPlaceUndefined represents an Activity that is stored in an undefined location
const ActivityPlaceUndefined ActivityPlace = 0

// ActivityPlaceInbox represents an Activity that is stored in a User's Inbox
const ActivityPlaceInbox ActivityPlace = 1

// ActivityPlaceOutbox represents an Activity that is stored in a User's Outbox
const ActivityPlaceOutbox ActivityPlace = 2

// ActivityPlaceFollowers represents an Activity that is stored in a User's Followers Outbox
const ActivityPlaceFollowers ActivityPlace = 3

// ActivityPlaceFollowing represents an Activity that is stored in a User's Following Outbox
const ActivityPlaceFollowing ActivityPlace = 4

// ActivityPlaceLiked represents an Activity that is stored in a User's Liked Outbox
const ActivityPlaceLiked ActivityPlace = 5

// ActivityPlaceKeys represents an Activity that is stored in a User's Key Outbox
const ActivityPlaceKeys ActivityPlace = 6

// ParseActivityPlace converts a string into an ActivityPlace
func ParseActivityPlace(value string) ActivityPlace {

	switch strings.ToLower(value) {
	case "inbox":
		return ActivityPlaceInbox
	case "outbox":
		return ActivityPlaceOutbox
	case "followers":
		return ActivityPlaceFollowers
	case "following":
		return ActivityPlaceFollowing
	case "liked":
		return ActivityPlaceLiked
	case "keys":
		return ActivityPlaceKeys
	default:
		return ActivityPlaceUndefined
	}
}

// String implements the Stringer interface
func (activityPlace ActivityPlace) String() string {

	switch activityPlace {
	case ActivityPlaceInbox:
		return "inbox"
	case ActivityPlaceOutbox:
		return "outbox"
	case ActivityPlaceFollowers:
		return "followers"
	case ActivityPlaceFollowing:
		return "following"
	case ActivityPlaceLiked:
		return "liked"
	case ActivityPlaceKeys:
		return "keys"
	default:
		return "undefined"
	}
}
