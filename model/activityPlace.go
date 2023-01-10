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

// ParseActivityPlace converts a string into an ActivityPlace
func ParseActivityPlace(value string) ActivityPlace {

	switch strings.ToLower(value) {
	case "inbox":
		return ActivityPlaceInbox
	case "outbox":
		return ActivityPlaceOutbox
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
	default:
		return "undefined"
	}
}
