package service

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestNotificationEmailLines pins what each row of the email says, for every Notification type
func TestNotificationEmailLines(t *testing.T) {

	test := func(notificationType string, wantAction string) {
		t.Run(notificationType, func(t *testing.T) {
			lines := notificationEmailLines([]model.Notification{emailTestNotification(notificationType, "Kyle", "Come with me")})
			require.Len(t, lines, 1)
			require.Equal(t, "Kyle", lines[0].Name)
			require.Equal(t, wantAction, lines[0].Action)
			require.Equal(t, "Come with me", lines[0].Summary)
			require.Equal(t, int64(1794200000000), lines[0].Date.UnixMilli())
		})
	}

	test(model.NotificationTypeDirect, "sent you a message")
	test(model.NotificationTypeMention, "mentioned you")
	test(model.NotificationTypeReply, "replied to you")
	test(model.NotificationTypeQuote, "quoted your post")
	test(model.NotificationTypeAnnounce, "shared your post")
	test(model.NotificationTypeLike, "liked your post")
	test(model.NotificationTypeDislike, "disliked your post")
	test(model.NotificationTypeFollow, "followed you")
	test("SOMETHING-NEW", "mentioned you")
}

// TestNotificationEmailLines_Fallbacks pins the encrypted label and the names used when one is missing
func TestNotificationEmailLines_Fallbacks(t *testing.T) {

	// RULE: An MLS message never prints its summary, even if one was stored
	encrypted := emailTestNotification(model.NotificationTypeDirect, "Kyle", "ciphertext that must not be shown")
	encrypted.Subtype = model.NotificationSubtypeMLS

	noName := emailTestNotification(model.NotificationTypeLike, "", "")
	noName.Actor.Username = "@kyle@remote.example"

	nobody := model.NewNotification()

	lines := notificationEmailLines([]model.Notification{encrypted, noName, nobody})

	require.Equal(t, "Encrypted message", lines[0].Summary)
	require.Equal(t, "@kyle@remote.example", lines[1].Name)
	require.Empty(t, lines[1].Summary)
	require.Equal(t, "Someone", lines[2].Name)

	require.Empty(t, notificationEmailLines(nil))
}

// TestUnreadNotificationsLabel pins the count that the subject line prints
func TestUnreadNotificationsLabel(t *testing.T) {
	require.Equal(t, "1 unread notification", unreadNotificationsLabel(1))
	require.Equal(t, "2 unread notifications", unreadNotificationsLabel(2))
	require.Equal(t, "2000 unread notifications", unreadNotificationsLabel(2000))
	require.Equal(t, "0 unread notifications", unreadNotificationsLabel(0))
}

// TestUserNotificationsEmail_Render renders the shipped email with an encrypted row, a summary
// that holds markup, and more rows than it lists
func TestUserNotificationsEmail_Render(t *testing.T) {

	email := loadEmbeddedEmail(t, "email-user-notifications", "user-notifications")

	encrypted := emailTestNotification(model.NotificationTypeDirect, "Kyle", "ciphertext that must not be shown")
	encrypted.Subtype = model.NotificationSubtypeMLS

	markup := emailTestNotification(model.NotificationTypeReply, "<b>Miles</b>", `<script>alert(1)</script><img src=x onerror=alert(2)>`)

	values := mapof.Any{
		"Email":            "sarah@connor.mil",
		"Name":             "Sarah",
		"CountLabel":       unreadNotificationsLabel(9),
		"Lines":            notificationEmailLines([]model.Notification{encrypted, markup}),
		"MoreCount":        int64(7),
		"NotificationsURL": "https://emissary.example/@me/notifications/all",
		"SettingsURL":      "https://emissary.example/@me/settings/notifications",
		"Domain_Name":      "Emissary Example",
		"Domain_Icon":      "",
	}

	var to, subject, body strings.Builder
	require.NoError(t, email.To.Execute(&to, values))
	require.NoError(t, email.Subject.Execute(&subject, values))
	require.NoError(t, email.Body.Execute(&body, values))

	require.Equal(t, "sarah@connor.mil", to.String())
	require.Equal(t, "9 unread notifications on Emissary Example", subject.String())

	html := body.String()

	// The encrypted row names its state and never its content
	require.Contains(t, html, "Encrypted message")
	require.NotContains(t, html, "ciphertext")

	// Text from another server is escaped
	require.NotContains(t, html, "<script>")
	require.NotContains(t, html, "<img src=x")
	require.NotContains(t, html, "<b>Miles</b>")
	require.Contains(t, html, "&lt;script&gt;")
	require.Contains(t, html, "&lt;b&gt;Miles&lt;/b&gt;")

	// Every link opens the All tab or the settings page
	require.Contains(t, html, `href="https://emissary.example/@me/notifications/all"`)
	require.Contains(t, html, `href="https://emissary.example/@me/settings/notifications"`)
	require.Contains(t, html, "And 7 more")
	require.NotContains(t, html, "no value")
	require.NotContains(t, html, "RULE:", "the template comment is not delivered")
}

// TestUserNotificationsEmail_NoMore pins that the count line is left out when every row is listed
func TestUserNotificationsEmail_NoMore(t *testing.T) {

	email := loadEmbeddedEmail(t, "email-user-notifications", "user-notifications")

	values := mapof.Any{
		"CountLabel": unreadNotificationsLabel(1),
		"Lines":      notificationEmailLines([]model.Notification{emailTestNotification(model.NotificationTypeFollow, "Kyle", "")}),
		"MoreCount":  int64(0),
	}

	var body strings.Builder
	require.NoError(t, email.Body.Execute(&body, values))
	require.NotContains(t, body.String(), "more &rarr;")
	require.Contains(t, body.String(), "followed you")
}

// TestUserNotificationsEmail_RequiredKeys pins that SendUnreadNotifications supplies every key
// that the "to" and subject templates read, since a missing one prints "<no value>"
func TestUserNotificationsEmail_RequiredKeys(t *testing.T) {

	service := testServerEmail()
	filesystem := os.DirFS(filepath.Join("..", "_embed", "templates", "email-user-notifications"))

	definition, err := os.ReadFile(filepath.Join("..", "_embed", "templates", "email-user-notifications", "email.hjson"))
	require.NoError(t, err)
	require.NoError(t, service.Add(filesystem, definition))

	require.Equal(t, []string{"CountLabel", "Email"}, []string(service.RequiredKeys("user-notifications")))
	require.NoError(t, service.RequireModel("user-notifications", "User"))
}

// TestNotificationRow_EveryType renders the shared row partial, as the minifier leaves it, for
// every Notification type, read and unread, and for an actor that the viewer's rules hide
func TestNotificationRow_EveryType(t *testing.T) {

	// The row calls one template that lives in base-social; a stub stands in for it here
	rows := template.Must(template.New("stub").Funcs(emissarytemplates.FuncMap(nullIconProvider{})).Parse(`{{define "label-chips"}}[chips]{{end}}`))
	filesystem := os.DirFS(filepath.Join("..", "_embed", "templates", "user-notifications"))
	require.NoError(t, loadHTMLTemplateFromFilesystem(filesystem, rows, emissarytemplates.FuncMap(nullIconProvider{})))

	render := func(t *testing.T, notification model.Notification) string {
		t.Helper()
		var result strings.Builder
		require.NoError(t, rows.ExecuteTemplate(&result, "row", notification))
		return result.String()
	}

	types := map[string]string{
		model.NotificationTypeDirect:   "sent you a message",
		model.NotificationTypeMention:  "mentioned you",
		model.NotificationTypeReply:    "replied to you",
		model.NotificationTypeQuote:    "quoted your post",
		model.NotificationTypeAnnounce: "shared your post",
		model.NotificationTypeLike:     "liked your post",
		model.NotificationTypeDislike:  "disliked your post",
		model.NotificationTypeFollow:   "followed you",
	}

	for notificationType, action := range types {
		t.Run(notificationType, func(t *testing.T) {

			unread := emailTestNotification(notificationType, "Kyle", "<i>Come with me</i>")
			html := render(t, unread)

			require.Contains(t, html, action)
			require.Contains(t, html, "Kyle")
			require.Contains(t, html, "&lt;i&gt;Come with me&lt;/i&gt;", "the summary is escaped")
			require.Contains(t, html, "read-one?notificationId="+unread.NotificationID.Hex(), "an unread row marks itself read")
			require.Contains(t, html, "unread")

			// A read row posts nothing
			read := unread
			read.ReadDate = 1234
			require.NotContains(t, render(t, read), "read-one")

			// A hidden actor is never named
			hidden := unread
			hidden.Labels = metadata.LabelSet{{Value: "Blocked", IsHidden: true}}
			require.True(t, hidden.Labels.IsHidden())
			require.NotContains(t, render(t, hidden), "Kyle")
		})
	}

	// Each type opens the place that §1.2 of NOTIFICATION-IMPROVEMENTS names
	t.Run("destinations", func(t *testing.T) {
		require.Contains(t, render(t, emailTestNotification(model.NotificationTypeDirect, "Kyle", "")), `href="/@me/conversations"`)
		require.Contains(t, render(t, emailTestNotification(model.NotificationTypeLike, "Kyle", "")), `href="https://remote.example/notes/1"`)
		require.Contains(t, render(t, emailTestNotification(model.NotificationTypeAnnounce, "Kyle", "")), `href="https://remote.example/notes/1"`)
		require.Contains(t, render(t, emailTestNotification(model.NotificationTypeFollow, "Kyle", "")), `hx-get="/@me/newsfeed/browse?url=https://remote.example/@kyle"`)
		require.Contains(t, render(t, emailTestNotification(model.NotificationTypeReply, "Kyle", "")), `hx-get="/@me/newsfeed/browse?url=https://remote.example/notes/1"`)

		quote := emailTestNotification(model.NotificationTypeQuote, "Kyle", "")
		require.Contains(t, render(t, quote), `hx-get="/@me/notifications/quote?notificationId=`+quote.NotificationID.Hex()+`"`)
	})

	// An encrypted message names its state and never its summary
	t.Run("encrypted", func(t *testing.T) {
		encrypted := emailTestNotification(model.NotificationTypeDirect, "Kyle", "ciphertext")
		encrypted.Subtype = model.NotificationSubtypeMLS
		html := render(t, encrypted)
		require.Contains(t, html, "Encrypted message")
		require.NotContains(t, html, "ciphertext")
	})
}

// emailTestNotification builds a Notification with the fields that the email and the row print
func emailTestNotification(notificationType string, name string, summary string) model.Notification {
	notification := model.NewNotification()
	notification.Type = notificationType
	notification.Actor.Name = name
	notification.Actor.ProfileURL = "https://remote.example/@" + strings.ToLower(name)
	notification.ObjectURL = "https://remote.example/notes/1"
	notification.ObjectSummary = summary
	notification.CreateDate = 1794200000000
	return notification
}
