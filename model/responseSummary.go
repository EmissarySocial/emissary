package model

// ResponseSummary collects the number of mentions, likes, and dislikes for a given object.
// It is embedded into other objects, such as Streams and Messages.
type ResponseSummary struct {
	ReplyCount   int `json:"replyCount,omitempty"   bson:"replyCount,omitempty"`   // Counter for the number of REPLIES for the containing object
	MentionCount int `json:"mentionCount,omitempty" bson:"mentionCount,omitempty"` // Counter for the number of MENTIONS for the containing object
	LikeCount    int `json:"likeCount,omitempty"    bson:"likeCount,omitempty"`    // Counter for the number of LIKES for the containing object
	DislikeCount int `json:"dislikeCount,omitempty" bson:"dislikeCount,omitempty"` // Counter for the number of DISLIKES for the containing object
}

func NewResponseSummary() ResponseSummary {
	return ResponseSummary{}
}

// HasReplies returns TRUE is this ResponseSummary has one or more Replies
func (summary ResponseSummary) HasReplies() bool {
	return summary.ReplyCount > 0
}

// HasMentions returns TRUE if this ResponseSummary has one or more Mentions
func (summary ResponseSummary) HasMentions() bool {
	return summary.MentionCount > 0
}

// HasLikes returns TRUE if this ResponseSummary has one or more Likes
func (summary ResponseSummary) HasLikes() bool {
	return summary.LikeCount > 0
}

// HasDislikes returns TRUE if this ResponseSummary has one or more Dislikes
func (summary ResponseSummary) HasDislikes() bool {
	return summary.DislikeCount > 0
}

func (summary ResponseSummary) CountByType(responseType string) int {
	switch responseType {

	case ResponseTypeReply:
		return summary.ReplyCount

	case ResponseTypeLike:
		return summary.LikeCount

	case ResponseTypeDislike:
		return summary.DislikeCount

	case ResponseTypeMention:
		return summary.MentionCount
	}

	return 0
}
