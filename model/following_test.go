package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestFollowingSchema(t *testing.T) {

	following := NewFollowing()
	s := schema.New(FollowingSchema())

	table := []tableTestItem{
		{"followingId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"folderId", "876543218765432187654321", nil},
		{"label", "LABEL", nil},
		{"username", "USERNAME", nil},
		{"url", "http://url.url", nil},
		{"profileUrl", "https://other.url", nil},
		{"iconUrl", "https://other.url/image.png", nil},
		{"behavior", "POSTS+REPLIES", nil},
		{"ruleAction", RuleActionMute, nil},
		{"isPublic", "true", true},
		{"method", FollowingMethodActivityPub, nil},
		{"status", FollowingStatusSuccess, nil},
		{"statusMessage", "STATUS-MESSAGE", nil},
		{"lastPolled", "123", int64(123)},
		{"pollDuration", "42", 42},
		{"nextPoll", 424242, int64(424242)},
		{"purgeDuration", "1", 1},
		{"errorCount", int64(7), 7},
	}

	tableTest_Schema(t, &s, &following, table)
}
