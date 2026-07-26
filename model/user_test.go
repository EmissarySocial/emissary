package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
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
		{"isBridgeBluesky", "true", true},
		{"isOwner", "true", true},
		{"isIndexable", "true", true},
		{"inboxTemplate", "INBOX", nil},
		{"outboxTemplate", "OUTBOX", nil},
		{"hashtags.0", "HEy", nil},
		{"hashtags.1", "ThErE", nil},
		{"hashtags.2", "bItChEs", nil},
		{"notificationChannels.0", NotificationChannelFollow, nil},
		{"notificationChannels.1", NotificationChannelReaction, nil},
		{"movedTo", "https://moved.example/newhome", nil},
		{"data.ABC", "DATA-ABC", nil},
		{"data.XYZ", "DATA-XYZ", nil},
		{"mapIds.federated", "fed-id-123", nil},
	}

	tableTest_Schema(t, &s, &user, tests)

	//TODO: Include DefaultAllow?

}

// TestUser_NotificationDefaults pins the locked design: new Users get the conversational
// channels (direct messages, mentions, replies) enabled, and ambient channels (followers,
// reactions) off.
func TestUser_NotificationDefaults(t *testing.T) {

	user := NewUser()

	require.Equal(t, sliceof.String{
		NotificationChannelDirectMessage,
		NotificationChannelMentionFollowing,
		NotificationChannelMentionNotFollowing,
		NotificationChannelReply,
	}, user.NotificationChannels)
}

// TestUser_NotificationEnabled pins the intersection test, including the locked rule
// that an EMPTY slice means "everything off".
func TestUser_NotificationEnabled(t *testing.T) {

	user := NewUser()

	// Defaults: mention/reply channels are on, follow/reaction are off
	require.True(t, user.NotificationEnabled([]string{NotificationChannelReply}))
	require.True(t, user.NotificationEnabled([]string{NotificationChannelMentionFollowing}))
	require.False(t, user.NotificationEnabled([]string{NotificationChannelFollow}))
	require.False(t, user.NotificationEnabled([]string{NotificationChannelReaction}))

	// "Either enables it": any overlapping channel is enough
	require.True(t, user.NotificationEnabled([]string{NotificationChannelReaction, NotificationChannelReply}))

	// Empty channels on the notification side match nothing
	require.False(t, user.NotificationEnabled(nil))

	// Empty settings mean everything off — no push, no dot, no SSE
	user.NotificationChannels = sliceof.String{}
	require.False(t, user.NotificationEnabled([]string{NotificationChannelReply}))
	require.False(t, user.NotificationEnabled([]string{NotificationChannelMentionFollowing}))
}

func TestUserJSONLD(t *testing.T) {
	user := NewUser()
	getter := any(user).(JSONLDGetter)
	require.NotNil(t, getter.GetJSONLD())
}

// TestUser_HashedPasswordAccessors pins the steranko.User contract: these are dumb
// accessors that store and return the value EXACTLY as given.  They must only ever
// receive already-hashed values (see service.SetPassword); if they gained any
// transformation logic, steranko's own hash writes would double-process.
func TestUser_HashedPasswordAccessors(t *testing.T) {
	user := NewUser()

	user.SetHashedPassword("$2a$12$not-really-a-hash-but-stored-verbatim")
	require.Equal(t, "$2a$12$not-really-a-hash-but-stored-verbatim", user.Password)
	require.Equal(t, "$2a$12$not-really-a-hash-but-stored-verbatim", user.GetHashedPassword())
}
