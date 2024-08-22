package model

import "go.mongodb.org/mongo-driver/bson/primitive"

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
	LastPolled  int64              `bson:"lastPolled"`
	NextPoll    int64              `bson:"nextPoll"`
	CreateDate  int64              `bson:"createDate"`
}

// FollowingSummaryFields returns a slice of all BSON field names for a FollowingSummary
func FollowingSummaryFields() []string {
	return []string{"_id", "username", "url", "label", "folder", "folderId", "iconUrl", "method", "status", "lastPolled", "nextPoll", "createDate"}
}

func (summary FollowingSummary) Fields() []string {
	return FollowingSummaryFields()
}

func (summary FollowingSummary) Icon() string {

	var icon string

	switch summary.Method {
	case FollowingMethodActivityPub:
		icon = "activitypub"
	case FollowingMethodPoll:
		icon = "rss"
	case FollowingMethodWebSub:
		icon = "websub"
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

func (summary FollowingSummary) GetRank() int64 {
	return summary.CreateDate
}

func (summary FollowingSummary) UsernameOrID() string {
	if summary.Username != "" {
		return summary.Username
	}
	return summary.URL
}
