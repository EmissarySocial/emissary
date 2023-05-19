package model

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

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
	}

	return "", false
}
