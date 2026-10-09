package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// FollowingSummary is an abbreviated Following, used when listing many Followings at once
type FollowingSummary struct {
	FollowingID primitive.ObjectID `json:"followingId" bson:"_id"`
	Username    string             `json:"username"    bson:"username"`
	URL         string             `json:"url"         bson:"url"`
	Label       string             `json:"label"       bson:"label"`
	Folder      string             `json:"folder"      bson:"folder"`
	FolderID    primitive.ObjectID `json:"folderId"    bson:"folderId"`
	IconURL     string             `json:"iconUrl"     bson:"iconUrl"`
	Method      string             `json:"method"      bson:"method"`
	Status      string             `json:"status"      bson:"status"`
	LastPolled  int64              `json:"lastPolled"  bson:"lastPolled"` // Unix epoch SECONDS when this Following was last polled (mirrors Following.LastPolled)
	NextPoll    int64              `json:"nextPoll"    bson:"nextPoll"`   // Unix epoch SECONDS when this Following is next due to be polled (mirrors Following.NextPoll)
	CreateDate  int64              `json:"createDate"  bson:"createDate"` // Unix epoch MILLISECONDS (journal projection; used only for sort rank)
}

// FollowingSummaryFields returns a slice of all BSON field names for a FollowingSummary
func FollowingSummaryFields() []string {
	return []string{"_id", "username", "url", "label", "folder", "folderId", "iconUrl", "method", "status", "lastPolled", "nextPoll", "createDate"}
}

// Fields returns the database fields required to populate a FollowingSummary
func (summary FollowingSummary) Fields() []string {
	return FollowingSummaryFields()
}

// Icon returns the name of the icon that represents this Following's status and polling method
func (summary FollowingSummary) Icon() string {

	// RULE: Every status the model paints red shows the SAME alert icon.  The three problem
	// states differ in how long we keep trying, never in how loudly they say something broke.
	if followingStatusClass(summary.Status) == "red" {
		return "alert-fill"
	}

	// A Following that is still connecting has nothing to report yet, so it spins
	if summary.Status == FollowingStatusLoading {
		return "loading"
	}

	// Name the protocol.  A Following that has not connected yet carries no Method, and an
	// empty name renders as an invisible icon, so fall back to a neutral account glyph.
	icon := "person"

	switch summary.Method {

	case FollowingMethodActivityPub:
		icon = "activitypub"

	case FollowingMethodPoll:
		icon = "rss"
	}

	// A working follow fills its protocol icon in
	if summary.Status == FollowingStatusSuccess {
		return icon + "-fill"
	}

	return icon
}

// StatusClass returns the CSS color suffix that represents this Following's current status,
// for use as "text-<class>"
func (summary FollowingSummary) StatusClass() string {
	return followingStatusClass(summary.Status)
}

// StatusLabel returns the human-readable description of this Following's current status
func (summary FollowingSummary) StatusLabel() string {
	return followingStatusLabel(summary.Status)
}

// StatusDescription returns one short sentence saying why this Following is in a problem status
func (summary FollowingSummary) StatusDescription() string {
	return followingStatusDescription(summary.Status)
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
