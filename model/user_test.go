package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestUserSchema confirms that every schema-exposed User property can be set and read
// through the schema round-trip.
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

// TestUserJSONLD confirms that User satisfies the JSONLDGetter interface
func TestUserJSONLD(t *testing.T) {
	user := NewUser()
	getter := any(user).(JSONLDGetter)
	require.NotNil(t, getter.GetJSONLD())
}

// TestUser_CalcProfileFingerprint pins the change-detection contract that drives profile
// federation (PROFILE-UPDATE-FEDERATION.md D-2): identical profiles hash identically, every
// field visible in GetJSONLD flips the fingerprint, and fields OUTSIDE the actor document
// (passwords, counters, email) never do — otherwise login-adjacent saves would spam
// followers with ActivityPub Updates.
func TestUser_CalcProfileFingerprint(t *testing.T) {

	newTestUser := func() User {
		user := NewUser()
		user.UserID = primitive.NilObjectID // pin the ID so every call builds the same document
		user.ProfileURL = "https://example.com/@000000000000000000000001"
		user.Username = "alice"
		user.DisplayName = "Alice"
		return user
	}

	baseline, err := newTestUser().CalcProfileFingerprint()
	require.Nil(t, err)
	require.Len(t, baseline, 64, "fingerprint must be a full hex SHA-256")

	// Stability: the same profile always produces the same fingerprint
	again, err := newTestUser().CalcProfileFingerprint()
	require.Nil(t, err)
	require.Equal(t, baseline, again)

	// changes asserts that a mutation ALTERS the fingerprint (the field is in the actor document)
	changes := func(name string, mutate func(*User)) {
		t.Helper()
		user := newTestUser()
		mutate(&user)
		result, err := user.CalcProfileFingerprint()
		require.Nil(t, err)
		require.NotEqual(t, baseline, result, "field %q should change the fingerprint", name)
	}

	// same asserts that a mutation does NOT alter the fingerprint (the field is not federated)
	same := func(name string, mutate func(*User)) {
		t.Helper()
		user := newTestUser()
		mutate(&user)
		result, err := user.CalcProfileFingerprint()
		require.Nil(t, err)
		require.Equal(t, baseline, result, "field %q must not change the fingerprint", name)
	}

	iconID, err := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.Nil(t, err)

	// Fields peers can see -> fingerprint changes -> an Update federates
	changes("displayName", func(u *User) { u.DisplayName = "Bob" })
	changes("username", func(u *User) { u.Username = "bob" })
	changes("statusMessage", func(u *User) { u.StatusMessage = "Hello, fediverse" })
	changes("profileUrl", func(u *User) { u.ProfileURL = "https://example.com/@000000000000000000000002" })
	changes("iconId", func(u *User) { u.IconID = iconID })
	changes("imageId", func(u *User) { u.ImageID = iconID })
	changes("hashtags", func(u *User) { u.Hashtags = sliceof.String{"golang"} })
	changes("links", func(u *User) {
		u.Links = sliceof.NewObject[PersonLink]()
		u.Links = append(u.Links, PersonLink{Name: "Blog", ProfileURL: "https://blog.example.com"})
	})
	changes("isIndexable", func(u *User) { u.IsIndexable = true })

	// Fields peers can NOT see -> fingerprint unchanged -> no Update on login-adjacent saves.
	// (location and isPublic are deliberate: neither appears in GetJSONLD today. isPublic only
	// affects the Update's ADDRESSING; if location is ever added to the actor document, move it
	// to the `changes` list above.)
	same("password", func(u *User) { u.Password = "$2a$12$hashhashhash" })
	same("passwordReset", func(u *User) { u.PasswordReset = PasswordReset{AuthCode: "abc123", ExpireDate: 999} })
	same("emailAddress", func(u *User) { u.EmailAddress = "alice@example.com" })
	same("followerCount", func(u *User) { u.FollowerCount = 42 })
	same("followingCount", func(u *User) { u.FollowingCount = 7 })
	same("ruleCount", func(u *User) { u.RuleCount = 3 })
	same("groupIds", func(u *User) { u.GroupIDs = append(u.GroupIDs, iconID) })
	same("location", func(u *User) { u.Location = "Underground Bunker" })
	same("isPublic", func(u *User) { u.IsPublic = true })
	same("profileFingerprint", func(u *User) { u.ProfileFingerprint = "feedface" })
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
