package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestUser_NotificationEmailWindow pins which createDates make a User owed an email
func TestUser_NotificationEmailWindow(t *testing.T) {

	const hour = millisecondsPerHour
	const now = 1000 * hour

	test := func(name string, hours int, watermark int64, emailAddress string, wantAfter int64, wantBefore int64, wantOK bool) {
		t.Run(name, func(t *testing.T) {
			user := NewUser()
			user.NotificationEmailHours = hours
			user.NotificationEmailDate = watermark
			user.EmailAddress = emailAddress

			after, before, ok := user.NotificationEmailWindow(now)
			require.Equal(t, wantOK, ok)
			require.Equal(t, wantAfter, after)
			require.Equal(t, wantBefore, before)
		})
	}

	test("24 hours, no watermark", 24, 0, "sarah@connor.mil", 0, now-24*hour, true)
	test("1 hour", 1, 0, "sarah@connor.mil", 0, now-hour, true)
	test("48 hours", 48, 0, "sarah@connor.mil", 0, now-48*hour, true)
	test("a watermark inside the span leaves a window", 6, now-10*hour, "sarah@connor.mil", now-10*hour, now-6*hour, true)
	test("no email", 0, 0, "sarah@connor.mil", 0, 0, false)
	test("no address", 24, 0, "", 0, 0, false)
	test("a span outside the list", 7, 0, "sarah@connor.mil", 0, 0, false)
	test("a negative span", -1, 0, "sarah@connor.mil", 0, 0, false)
	test("a watermark newer than the span", 24, now-hour, "sarah@connor.mil", 0, 0, false)
	test("a watermark at the edge of the span", 24, now-24*hour, "sarah@connor.mil", 0, 0, false)
	test("a watermark in the future", 24, now+hour, "sarah@connor.mil", 0, 0, false)
}

// TestNewUser_NotificationEmailDefault pins that a new User gets the email after 24 hours
func TestNewUser_NotificationEmailDefault(t *testing.T) {
	user := NewUser()
	require.Equal(t, 24, user.NotificationEmailHours)
	require.Zero(t, user.NotificationEmailDate)
	require.Contains(t, NotificationEmailHours(), NotificationEmailHoursDefault)
}

// TestUser_SetNotificationEmailHours pins that only a switch from No email starts the clock
func TestUser_SetNotificationEmailHours(t *testing.T) {

	user := NewUser()
	user.NotificationEmailDate = 100

	// Changing one span for another keeps the watermark
	user.setNotificationEmailHours(6, 500)
	require.Equal(t, 6, user.NotificationEmailHours)
	require.Equal(t, int64(100), user.NotificationEmailDate)

	// Posting the same value again, as the settings form does on every save, changes nothing
	user.setNotificationEmailHours(6, 600)
	require.Equal(t, int64(100), user.NotificationEmailDate)

	// Switching off keeps the watermark
	user.setNotificationEmailHours(0, 700)
	require.Zero(t, user.NotificationEmailHours)
	require.Equal(t, int64(100), user.NotificationEmailDate)

	// Switching back on starts the clock at that moment
	user.setNotificationEmailHours(12, 800)
	require.Equal(t, 12, user.NotificationEmailHours)
	require.Equal(t, int64(800), user.NotificationEmailDate)
}

// TestUserSchema_NotificationEmailHours pins that the schema accepts the listed spans and no other
func TestUserSchema_NotificationEmailHours(t *testing.T) {

	userSchema := schema.New(UserSchema())

	for _, hours := range NotificationEmailHours() {
		user := NewUser()
		require.NoError(t, userSchema.Set(&user, "notificationEmailHours", hours))
		require.Equal(t, hours, user.NotificationEmailHours)

		value, err := userSchema.Get(&user, "notificationEmailHours")
		require.NoError(t, err)
		require.Equal(t, hours, value)
	}

	// The settings form posts strings
	user := NewUser()
	require.NoError(t, userSchema.Set(&user, "notificationEmailHours", "48"))
	require.Equal(t, 48, user.NotificationEmailHours)

	// A span outside the list is refused
	element, isInteger := UserSchema().(schema.Object).Properties["notificationEmailHours"].(schema.Integer)
	require.True(t, isInteger)
	require.NoError(t, element.Validate(24))
	require.Error(t, element.Validate(7))

	// No form can write the watermark
	require.Error(t, userSchema.Set(&user, "notificationEmailDate", 12345))
}

// TestUser_NotificationEmailFieldsStayOutOfJSON pins that the watermark is never serialized
func TestUser_NotificationEmailFieldsStayOutOfJSON(t *testing.T) {
	user := NewUser()
	user.NotificationEmailDate = 987654321
	require.NotContains(t, user.GetJSONLD(), "notificationEmailDate")
	require.NotContains(t, user.GetJSONLD(), "notificationEmailHours")
}
