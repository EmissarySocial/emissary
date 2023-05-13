package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

// ResponseSummary collects the number of mentions, likes, and dislikes for a given object.
// It is embedded into other objects, such as Streams and Messages.
type ResponseSummary struct {
	ReplyCount   int `json:"replyCount,omitempty" bson:"replyCount,omitempty"`     // Counter for the number of REPLIES for the containing object
	MentionCount int `json:"mentionCount,omitempty" bson:"mentionCount,omitempty"` // Counter for the number of MENTIONS for the containing object
	LikeCount    int `json:"likeCount,omitempty"    bson:"likeCount,omitempty"`    // Counter for the number of LIKES for the containing object
	DislikeCount int `json:"dislikeCount,omitempty" bson:"dislikeCount,omitempty"` // Counter for the number of DISLIKES for the containing object
}

func NewResponseSummary() ResponseSummary {
	return ResponseSummary{}
}

func ResponseSummarySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"replyCount":   schema.Integer{Minimum: null.NewInt64(0)},
			"mentionCount": schema.Integer{Minimum: null.NewInt64(0)},
			"likeCount":    schema.Integer{Minimum: null.NewInt64(0)},
			"dislikeCount": schema.Integer{Minimum: null.NewInt64(0)},
		},
	}
}

func (summary *ResponseSummary) GetPointer(name string) (any, bool) {

	switch name {

	case "replyCount":
		return &summary.ReplyCount, true

	case "mentionCount":
		return &summary.MentionCount, true

	case "likeCount":
		return &summary.LikeCount, true

	case "dislikeCount":
		return &summary.DislikeCount, true

	default:
		return "", false
	}
}
