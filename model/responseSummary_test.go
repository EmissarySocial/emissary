package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestResponseSummary(t *testing.T) {

	signupForm := NewResponseSummary()

	s := schema.New(ResponseSummarySchema())

	table := []tableTestItem{
		{"replyCount", 42, nil},
		{"mentionCount", 42, nil},
		{"likeCount", 42, nil},
		{"dislikeCount", 42, nil},
	}

	tableTest_Schema(t, &s, &signupForm, table)
}
