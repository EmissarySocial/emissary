package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestUserSchema(t *testing.T) {

	s := schema.New(UserSchema())
	user := NewUser()

	tests := []tableTestItem{
		{"userId", "000000000000000000000001", nil},
		{"groupIds.0", "000000000000000000000002", nil},
		{"groupIds.1", "000000000000000000000003", nil},
		{"groupIds.2", "000000000000000000000004", nil},
		{"iconId", "000000000000000000000005", nil},
		{"imageId", "000000000000000000000006", nil},
		{"displayName", "USER", nil},
		{"statusMessage", "STATUS", nil},
		{"location", "LOCATION", nil},
		{"links.0.name", "LINK 1", nil},
		{"links.0.profileUrl", "https://profile.url", nil},
		{"profileUrl", "http://profile.url", nil},
		{"emailAddress", "email@address.url", nil},
		{"username", "USERNAME", nil},
		{"locale", "en-us", nil},
		{"stateId", "STATE", nil},
		{"signupNote", "LetMeInBro", nil},
		{"followerCount", "1", 1},
		{"followingCount", "2", 2},
		{"ruleCount", "3", 3},
		{"isPublic", "true", true},
		{"isOwner", "true", true},
		{"isIndexable", "true", true},
		{"inboxTemplate", "INBOX", nil},
		{"outboxTemplate", "OUTBOX", nil},
		{"hashtags.0", "HEy", nil},
		{"hashtags.1", "ThErE", nil},
		{"hashtags.2", "bItChEs", nil},
	}

	tableTest_Schema(t, &s, &user, tests)

	//TODO: Include DefaultAllow?

}

func TestUserJSONLD(t *testing.T) {
	user := NewUser()
	getter := any(user).(JSONLDGetter)
	require.NotNil(t, getter.GetJSONLD())
}
