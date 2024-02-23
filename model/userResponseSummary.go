package model

import "github.com/benpate/hannibal/vocab"

// UserResponseSummary is a summary object of the User's response(s) to a particular ActivityStream
type UserResponseSummary struct {
	Announce bool
	Like     bool
	Dislike  bool
}

// NewUserResponseSummary returns a fully initialized UserResponseSummary
func NewUserResponseSummary() UserResponseSummary {
	return UserResponseSummary{}
}

func (summary *UserResponseSummary) SetResponse(responseType string, value bool) {
	switch responseType {

	case vocab.ActivityTypeAnnounce:
		summary.Announce = value

	case vocab.ActivityTypeLike:
		summary.Like = value

	case vocab.ActivityTypeDislike:
		summary.Dislike = value
	}
}
