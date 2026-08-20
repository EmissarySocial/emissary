package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// FollowingSummary is an abbreviated Following, used when listing many Followings at once
type FollowingSummary struct {
	FollowingID primitive.ObjectID `bson:"_id"`
	Username    string             `bson:"username"`
	URL         string             `bson:"url"`
	Label       string             `bson:"label"`
	Folder      string             `bson:"folder"`
	FolderID    primitive.ObjectID `bson:"folderId"`
	IconURL     string             `bson:"iconUrl"`
	Method      string             `bson:"method"`
	Status      string             `bson:"status"`
	LastPolled  int64              `bson:"lastPolled"` // Unix epoch SECONDS when this Following was last polled (mirrors Following.LastPolled)
	NextPoll    int64              `bson:"nextPoll"`   // Unix epoch SECONDS when this Following is next due to be polled (mirrors Following.NextPoll)
	CreateDate  int64              `bson:"createDate"` // Unix epoch MILLISECONDS (journal projection; used only for sort rank)
}

// FollowingSummaryFields returns a slice of all BSON field names for a FollowingSummary
func FollowingSummaryFields() []string {
	return []string{"_id", "username", "url", "label", "folder", "folderId", "iconUrl", "method", "status", "lastPolled", "nextPoll", "createDate"}
}

// Fields returns the database fields required to populate a FollowingSummary
func (summary FollowingSummary) Fields() []string {
	return FollowingSummaryFields()
}

// Icon returns the name of the icon that represents this Following's polling method
func (summary FollowingSummary) Icon() string {

	var icon string

	switch summary.Method {

	case FollowingMethodActivityPub:
		icon = "activitypub"

	case FollowingMethodPoll:
		icon = "rss"
	}

	switch summary.Status {

	case FollowingStatusLoading:
		return "loading"

	case FollowingStatusSuccess:
		return icon + "-fill"

	default:
		return icon
	}
}

// StatusClass returns the CSS class that represents this Following's current status
func (summary FollowingSummary) StatusClass() string {

	switch summary.Status {

	case FollowingStatusLoading:
		return "spin"

	case FollowingStatusFailure:
		return "red"

	case FollowingStatusSuccess:
		return "green"

	default:
		return ""
	}
}

// GetRank returns the sort rank of this FollowingSummary
func (summary FollowingSummary) GetRank() int64 {
	return summary.CreateDate
}

// UsernameOrID returns this Following's username, falling back to its URL
func (summary FollowingSummary) UsernameOrID() string {
	if summary.Username != "" {
		return summary.Username
	}
	return summary.URL
}
