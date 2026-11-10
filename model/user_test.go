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
		{"isPrimary", "true", true},
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

// TestUser_NotificationDefaults pins the locked design for a new User's channels
func TestUser_NotificationDefaults(t *testing.T) {

	// Conversational channels (direct messages, mentions, replies) start enabled;
	// ambient ones (followers, reactions) start off.
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

// TestUser_CalcProfileFingerprint pins the change-detection contract that drives
// profile federation (PROFILE-UPDATE-FEDERATION.md D-2)
func TestUser_CalcProfileFingerprint(t *testing.T) {

	// Identical profiles hash identically, every field visible in GetJSONLD flips the
	// fingerprint, and fields outside the actor document never do -- otherwise
	// login-adjacent saves would spam followers with ActivityPub Updates.
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

	// Fields peers can NOT see -> fingerprint unchanged -> no Update on login-adjacent
	// saves. location and isPublic are deliberate: neither appears in GetJSONLD today,
	// and isPublic affects only the Update's ADDRESSING.
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

// TestUser_HashedPasswordAccessors pins the steranko.User contract: dumb accessors
// that store and return the value EXACTLY as given
func TestUser_HashedPasswordAccessors(t *testing.T) {

	// They must only ever receive already-hashed values (see service.SetPassword). Any
	// transformation here would make steranko's own hash writes double-process.
	user := NewUser()

	user.SetHashedPassword("$2a$12$not-really-a-hash-but-stored-verbatim")
	require.Equal(t, "$2a$12$not-really-a-hash-but-stored-verbatim", user.Password)
	require.Equal(t, "$2a$12$not-really-a-hash-but-stored-verbatim", user.GetHashedPassword())
}

// TestUser_RepresentsDomain verifies that only a live owner flagged primary represents the domain
func TestUser_RepresentsDomain(t *testing.T) {

	newUser := func(isOwner bool, isPrimary bool) User {
		user := NewUser()
		user.IsOwner = isOwner
		user.IsPrimary = isPrimary
		return user
	}

	require.True(t, newUser(true, true).RepresentsDomain())
	require.False(t, newUser(true, false).RepresentsDomain(), "an owner who is not primary")
	require.False(t, newUser(false, true).RepresentsDomain(), "a primary flag on a User who is not an owner")
	require.False(t, newUser(false, false).RepresentsDomain())
	require.False(t, User{}.RepresentsDomain(), "an empty User")

	// RULE: A deleted User represents nobody, even with both flags set
	deleted := newUser(true, true)
	deleted.SetDeleted("test")
	require.False(t, deleted.RepresentsDomain())
}

// TestUser_IndieAuthMe verifies that only the public primary user gets a `me`
func TestUser_IndieAuthMe(t *testing.T) {

	const profileURL = "https://example.com/@000000000000000000000001"

	newUser := func(isPrimary bool, isPublic bool) User {
		user := NewUser()
		user.IsOwner = true
		user.IsPrimary = isPrimary
		user.IsPublic = isPublic
		user.ProfileURL = profileURL
		return user
	}

	require.Equal(t, profileURL, newUser(true, true).IndieAuthMe())
	require.Empty(t, newUser(true, false).IndieAuthMe(), "a non-public profile answers 404, so no client could confirm it")
	require.Empty(t, newUser(false, true).IndieAuthMe(), "an owner who is not primary")
	require.Empty(t, User{}.IndieAuthMe())

	demoted := newUser(true, true)
	demoted.IsOwner = false
	require.Empty(t, demoted.IndieAuthMe(), "a primary flag on a User who is not an owner")

	deleted := newUser(true, true)
	deleted.SetDeleted("test")
	require.Empty(t, deleted.IndieAuthMe())
}

// TestUser_SummaryCarriesPrimary verifies that the summary the admin user list reads names the primary user
func TestUser_SummaryCarriesPrimary(t *testing.T) {

	user := NewUser()
	require.False(t, user.Summary().IsPrimary)

	user.IsPrimary = true
	require.True(t, user.Summary().IsPrimary)
	require.Contains(t, UserSummaryFields(), "isPrimary")
}

// TestUser_PrimaryStaysOutOfTheActor verifies that the primary flag is not published, so changing it
// federates no profile Update
func TestUser_PrimaryStaysOutOfTheActor(t *testing.T) {

	user := NewUser()
	user.Username = "sarah"
	before, err := user.CalcProfileFingerprint()
	require.NoError(t, err)

	user.IsOwner = true
	user.IsPrimary = true
	after, err := user.CalcProfileFingerprint()
	require.NoError(t, err)

	require.Equal(t, before, after)
}
