package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// newSocialUserService returns a User service whose Template service holds one outbox
// Template, loaded from hjson the way Template.Add loads it
func newSocialUserService(t *testing.T, definition string) *User {
	t.Helper()

	template := model.NewTemplate("social-outbox", nil)
	require.NoError(t, hjson.Unmarshal([]byte(definition), &template))
	template.Schema.Inherit(schema.New(template.BaseSchema()))

	templateService := &Template{templates: set.Map[model.Template]{"social-outbox": template}}
	return &User{host: "https://example.com", templateService: templateService}
}

// newSocialUser returns a User whose outbox Template is the one newSocialUserService loads
func newSocialUser() model.User {
	user := model.NewUser()
	user.Username = "alice"
	user.ProfileURL = "https://example.com/@" + user.UserID.Hex()
	user.OutboxTemplate = "social-outbox"
	return user
}

// TestUser_ApplySocialRules covers an outbox Template with rules, one without, a missing
// Template, and a rule that fails
func TestUser_ApplySocialRules(t *testing.T) {

	t.Run("the outbox Template's rules replace url with a list of Links", func(t *testing.T) {

		// FUNKWHALE task 1.2: a Funkwhale channel lists an RSS Link in its url
		service := newSocialUserService(t, `{
			model: User
			socialRules: [
				{target:"url", value:[]}
				{target:"url.0.type", value:"Link"}
				{target:"url.0.mediaType", value:"text/html"}
				{target:"url.0.href", expression:"{{.Host}}/@{{.Username}}"}
				{target:"url.1.type", value:"Link"}
				{target:"url.1.mediaType", value:"application/rss+xml"}
				{target:"url.1.href", expression:"{{.Host}}/@{{.Username}}/feed?format=rss"}
				{target:"summary", path:"username"}
			]
		}`)

		user := newSocialUser()
		result := user.GetJSONLD()
		service.applySocialRules(&user, &result)

		require.Equal(t, sliceof.Any{
			mapof.Any{"type": "Link", "mediaType": "text/html", "href": "https://example.com/@alice"},
			mapof.Any{"type": "Link", "mediaType": "application/rss+xml", "href": "https://example.com/@alice/feed?format=rss"},
		}, result["url"])
		require.Equal(t, "alice", result["summary"], "path rules read the User through UserSchema")
		require.Equal(t, user.ActivityPubURL(), result["id"], "every other property is unchanged")
	})

	t.Run("a Template with no rules changes nothing", func(t *testing.T) {
		service := newSocialUserService(t, `{model: "User"}`)

		user := newSocialUser()
		result := user.GetJSONLD()
		service.applySocialRules(&user, &result)
		require.Equal(t, user.GetJSONLD(), result)
	})

	t.Run("a missing Template changes nothing", func(t *testing.T) {
		service := newSocialUserService(t, `{model: "User"}`)

		user := newSocialUser()
		user.OutboxTemplate = "missing"
		result := user.GetJSONLD()
		service.applySocialRules(&user, &result)
		require.Equal(t, user.GetJSONLD(), result)
	})

	t.Run("a failed rule keeps the output of the rules before it", func(t *testing.T) {
		service := newSocialUserService(t, `{
			model: User
			socialRules: [
				{target:"summary", value:"before"}
				{target:"name.first", value:"a string cannot hold a key"}
				{target:"after", value:"never written"}
			]
		}`)

		user := newSocialUser()
		user.DisplayName = "Alice"
		result := user.GetJSONLD()
		service.applySocialRules(&user, &result)

		require.Equal(t, "before", result["summary"])
		require.Equal(t, "Alice", result["name"])
		require.NotContains(t, result, "after")
	})
}
