package build

import (
	"crypto/sha256"
	"encoding/base64"
	"html"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestReportedBug_OAuthAuthorizePageNo500 verifies that the shared head template renders
// against the OAuth builder, and marks the consent page "noindex"
func TestReportedBug_OAuthAuthorizePageNo500(t *testing.T) {

	// GET /oauth/authorize once answered 500: includes-head evaluates .IsIndexable, which the
	// builder lacked until it embedded Theme. This is the literal conditional from includes-head.
	head := template.Must(template.New("head").Parse(
		`{{- if not .IsIndexable }}<meta name="robots" content="noindex">{{- end }}`))

	// The reported bug: executing the head against the OAuth builder must not error...
	var buffer strings.Builder
	require.Nil(t, head.Execute(&buffer, OAuthAuthorization{}))

	// ...and the consent page is ephemeral + user-specific, so it must emit "noindex".
	require.Equal(t, `<meta name="robots" content="noindex">`, buffer.String())
	require.False(t, OAuthAuthorization{}.IsIndexable())
}

// TestOAuthAuthorization_PKCEAccessors verifies that the consent page reads the PKCE
// challenge from the request being authorized, and reads nothing when there is none.
func TestOAuthAuthorization_PKCEAccessors(t *testing.T) {

	builder := OAuthAuthorization{_transaction: model.OAuthAuthorizationRequest{
		CodeChallenge:       "challenge",
		CodeChallengeMethod: model.PKCEMethodS256,
	}}

	require.Equal(t, "challenge", builder.CodeChallenge())
	require.Equal(t, model.PKCEMethodS256, builder.CodeChallengeMethod())

	require.Equal(t, "", OAuthAuthorization{}.CodeChallenge())
	require.Equal(t, "", OAuthAuthorization{}.CodeChallengeMethod())
}

// TestReportedBug_ConsentFormPostsPKCE verifies that the consent page posts back the PKCE
// challenge it was shown, so the approved code is bound to the client's verifier
func TestReportedBug_ConsentFormPostsPKCE(t *testing.T) {

	// BUG-361: the form posted no PKCE fields, so Approve failed for every client without a
	// secret, and a confidential client's code silently lost its challenge.
	// RFC 7636 Appendix B's verifier, and the S256 challenge computed from it
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])

	const redirectURI = `https://client.example/callback?x=1&y=2`

	newClient := func(secret string) model.OAuthClient {
		client := model.NewOAuthClient()
		client.ClientSecret = secret
		client.RedirectURIs = append(client.RedirectURIs, redirectURI)
		client.Scopes = append(client.Scopes, "read", "write")
		return client
	}

	// The query string a client sends to GET /oauth/authorize, leaving out any PKCE value that is empty
	query := func(codeChallenge string, method string) url.Values {
		result := url.Values{
			"client_id":     {"https://client.example/client.json"},
			"redirect_uri":  {redirectURI},
			"response_type": {"code"},
			"scope":         {"read write"},
			"state":         {`a&b="c"<d>`},
			"me":            {"https://example.com/@someone-else"},
		}

		if codeChallenge != "" {
			result.Set("code_challenge", codeChallenge)
		}

		if method != "" {
			result.Set("code_challenge_method", method)
		}

		return result
	}

	// A code bound to the posted challenge redeems with the client's verifier, and only with it
	redeems := func(t *testing.T, posted model.OAuthAuthorizationRequest) {
		t.Helper()
		grant := model.NewOAuthUserToken()
		grant.SetPKCEChallenge(posted.CodeChallenge, posted.CodeChallengeMethod)
		require.True(t, grant.HasPKCEChallenge())
		require.NoError(t, grant.VerifyPKCE(verifier))
		require.Error(t, grant.VerifyPKCE(strings.Repeat("x", 43)))
		require.Error(t, grant.VerifyPKCE(""))
	}

	t.Run("public client, S256", func(t *testing.T) {
		client := newClient("")
		posted := consentRoundTrip(t, client, query(challenge, model.PKCEMethodS256))
		require.Equal(t, challenge, posted.CodeChallenge)
		require.Equal(t, model.PKCEMethodS256, posted.CodeChallengeMethod)
		require.NoError(t, posted.Validate(client))
		redeems(t, posted)
	})

	t.Run("public client, plain", func(t *testing.T) {
		client := newClient("")
		posted := consentRoundTrip(t, client, query(challenge, model.PKCEMethodPlain))
		require.Equal(t, model.PKCEMethodPlain, posted.CodeChallengeMethod)
		require.NoError(t, posted.Validate(client))
	})

	// RFC 7636 §4.3: a challenge sent without a method is "plain", and the page posts it as such
	t.Run("public client, method omitted", func(t *testing.T) {
		client := newClient("")
		posted := consentRoundTrip(t, client, query(challenge, ""))
		require.Equal(t, model.PKCEMethodPlain, posted.CodeChallengeMethod)
		require.NoError(t, posted.Validate(client))
	})

	// The silent half of the bug: a confidential client was approved, but its code was not bound
	t.Run("confidential client keeps its binding", func(t *testing.T) {
		client := newClient("secret")
		posted := consentRoundTrip(t, client, query(challenge, model.PKCEMethodS256))
		require.Equal(t, challenge, posted.CodeChallenge)
		require.NoError(t, posted.Validate(client))
		redeems(t, posted)
	})

	// A confidential client that does not use PKCE posts empty fields, which bind nothing
	t.Run("confidential client without PKCE", func(t *testing.T) {
		client := newClient("secret")
		posted := consentRoundTrip(t, client, query("", ""))
		require.Equal(t, "", posted.CodeChallenge)
		require.Equal(t, "", posted.CodeChallengeMethod)
		require.NoError(t, posted.Validate(client))

		grant := model.NewOAuthUserToken()
		grant.SetPKCEChallenge(posted.CodeChallenge, posted.CodeChallengeMethod)
		require.False(t, grant.HasPKCEChallenge())
	})
}

// consentRoundTrip binds an authorization request as GetOAuthAuthorization does, renders
// the consent page for it, and returns the page's hidden inputs bound as PostOAuthAuthorization binds them
func consentRoundTrip(t *testing.T, client model.OAuthClient, query url.Values) model.OAuthAuthorizationRequest {

	t.Helper()

	// GET: bind the query string, then validate it against the client, as NewOAuthAuthorization does
	shown := model.NewOAuthAuthorizationRequest()
	getRequest := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query.Encode(), nil)
	require.NoError(t, echo.New().NewContext(getRequest, httptest.NewRecorder()).Bind(&shown))
	require.NoError(t, shown.Validate(client))

	// Render the consent page
	user := model.NewUser()
	builder := OAuthAuthorization{
		_client:      client,
		_transaction: shown,
		_user:        &user,
		Theme:        Theme{Common: Common{_factory: newTestDomainFactory(mapof.NewAny(), mapof.NewString())}},
	}

	var page strings.Builder
	require.NoError(t, parseConsentPage(t).ExecuteTemplate(&page, "oauth", builder))
	require.Regexp(t, `<form action="/oauth/authorize" method="post"`, page.String())

	// POST: bind only what the page's own form carries
	postRequest := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(consentHiddenInputs(page.String()).Encode()))
	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	posted := model.NewOAuthAuthorizationRequest()
	require.NoError(t, echo.New().NewContext(postRequest, httptest.NewRecorder()).Bind(&posted))

	// The page posts back everything it was shown, with the client named by its own ID.  The `me` hint is
	// only shown, because the grant goes to whoever is signed in.
	require.NotEmpty(t, shown.Me)
	expected := shown
	expected.ClientID = client.ClientID.Hex()
	expected.Me = ""
	require.Equal(t, expected, posted)

	return posted
}

// parseConsentPage parses theme-global's consent template, with its shared partials left empty
func parseConsentPage(t *testing.T) *template.Template {

	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "_embed", "templates", "theme-global", "oauth.html"))
	require.NoError(t, err)

	set := template.New("").Funcs(templates.FuncMap(service.NewIcons()))
	template.Must(set.New("includes-head").Parse(""))
	template.Must(set.New("includes-foot").Parse(""))
	template.Must(set.New("oauth").Parse(string(content)))

	return set
}

// consentHiddenInputs returns the name and value of every hidden input on a page, as a browser would post them
func consentHiddenInputs(page string) url.Values {

	result := url.Values{}
	nameAttribute := regexp.MustCompile(`name="([^"]*)"`)
	valueAttribute := regexp.MustCompile(`value="([^"]*)"`)

	for _, tag := range regexp.MustCompile(`<input[^>]*>`).FindAllString(page, -1) {

		if !strings.Contains(tag, `type="hidden"`) {
			continue
		}

		name := nameAttribute.FindStringSubmatch(tag)
		if name == nil {
			continue
		}

		value := ""
		if match := valueAttribute.FindStringSubmatch(tag); match != nil {
			value = html.UnescapeString(match[1])
		}

		result.Add(html.UnescapeString(name[1]), value)
	}

	return result
}

// TestOAuthAuthorization_ClientDetails verifies the accessors that tell the User who is asking and where they go next
func TestOAuthAuthorization_ClientDetails(t *testing.T) {

	builder := OAuthAuthorization{
		_client:      model.OAuthClient{ClientURL: "https://app.example/"},
		_transaction: model.OAuthAuthorizationRequest{RedirectURI: model.OAuthRedirectURIOutOfBand, Me: "https://example.com/"},
	}

	require.Equal(t, "https://app.example/", builder.ClientURL())
	require.True(t, builder.IsOutOfBand())
	require.Equal(t, "https://example.com/", builder.Me())

	// A registered client that redirects has no URL of its own and is not out of band
	require.Equal(t, "", OAuthAuthorization{}.ClientURL())
	require.False(t, OAuthAuthorization{}.IsOutOfBand())
}

// TestOAuthAuthorization_SwitchAccountURL verifies that switching accounts returns to the whole request, query included
func TestOAuthAuthorization_SwitchAccountURL(t *testing.T) {

	require.Equal(t, "/signout", OAuthAuthorization{}.SwitchAccountURL())

	const requestURI = "/oauth/authorize?client_id=https%3A%2F%2Fapp.example%2F&scope=read%20write&state=a%26b"
	request := httptest.NewRequest(http.MethodGet, requestURI, nil)
	builder := OAuthAuthorization{Theme: Theme{Common: Common{_request: request}}}

	switchURL, err := url.Parse(builder.SwitchAccountURL())
	require.NoError(t, err)
	require.Equal(t, "/signout", switchURL.Path)
	require.Equal(t, requestURI, switchURL.Query().Get("next"))
}

// TestDescribeScopes verifies that every requested scope is listed once, in plain words where the dataset has them
func TestDescribeScopes(t *testing.T) {

	// values returns the scopes a request describes, split exactly as the grant splits them
	values := func(scope string) []string {
		result := make([]string, 0)
		for _, lookupCode := range describeScopes(model.OAuthAuthorizationRequest{Scope: scope}.Scopes()) {
			result = append(result, lookupCode.Value)
		}
		return result
	}

	// An empty scope requests only the User's identity
	require.Empty(t, values(""))
	require.Empty(t, values("   "))

	// The dataset's order wins, so each group is one run
	require.Equal(t, []string{"profile", "read:statuses", "write"}, values("write profile read:statuses"))

	// Each scope is listed once, and empty ones are dropped
	require.Equal(t, []string{"read", "write"}, values("read  read write"))

	// Scopes the dataset does not describe follow, by name, in the order requested
	require.Equal(t, []string{"read", "create", "made-up"}, values("create read made-up create"))

	// A tab does not separate scopes, here or on the grant, so the page shows what is stored
	require.Equal(t, []string{"read\twrite"}, values("read\twrite"))

	// An unknown scope is named, never described as anything it is not
	other := describeScopes([]string{"create"})
	require.Equal(t, []form.LookupCode{{Value: "create", Label: "create", Description: "A permission this server does not recognize.", Icon: "key", Group: "Other"}}, other)

	// A known scope carries the dataset's own words
	known := describeScopes([]string{"email"})
	require.Len(t, known, 1)
	require.Equal(t, "Your Email Address", known[0].Label)
	require.Equal(t, "Identity", known[0].Group)
}

// FuzzDescribeScopes requires every requested scope exactly once, with the described ones ahead of the rest
func FuzzDescribeScopes(f *testing.F) {

	f.Add("")
	f.Add("read write follow push")
	f.Add("profile email")
	f.Add("read  read\twrite made-up")
	f.Add("activitypub_account_portability create admin:write")

	f.Fuzz(func(t *testing.T, scope string) {

		requested := model.OAuthAuthorizationRequest{Scope: scope}.Scopes()

		expected := make(map[string]bool)
		for _, value := range requested {
			if value != "" {
				expected[value] = true
			}
		}

		seen := make(map[string]bool)
		reachedOther := false

		for _, lookupCode := range describeScopes(requested) {

			require.False(t, seen[lookupCode.Value], "scope %q is listed twice", lookupCode.Value)
			require.True(t, expected[lookupCode.Value], "scope %q was never requested", lookupCode.Value)
			seen[lookupCode.Value] = true

			if lookupCode.Group == "Other" {
				reachedOther = true
				continue
			}

			require.False(t, reachedOther, "described scope %q follows an unknown one", lookupCode.Value)
		}

		require.Len(t, seen, len(expected))
	})
}

// TestSignedInAsSomeoneElse verifies which `me` hints warn that a different User is signed in
func TestSignedInAsSomeoneElse(t *testing.T) {

	alice := model.NewUser()
	alice.Username = "alice"

	names := func(me string) bool {
		return signedInAsSomeoneElse(me, "example.com", &alice)
	}

	// Another User's profile on this host
	require.True(t, names("https://example.com/@bob"))
	require.True(t, names("https://example.com/@bob/"))
	require.True(t, names("https://EXAMPLE.com/@bob"))
	require.True(t, names("http://example.com:8080/@bob"))
	require.True(t, names("https://example.com/@"+primitiveID()))

	// The signed-in User, by username in any case, by ID, or as "me"
	require.False(t, names("https://example.com/@alice"))
	require.False(t, names("https://example.com/@ALICE"))
	require.False(t, names("https://example.com/@"+alice.UserID.Hex()))
	require.False(t, names("https://example.com/@me"))

	// The domain root, another host, and URLs that are not a profile name nobody to switch to
	require.False(t, names(""))
	require.False(t, names("https://example.com"))
	require.False(t, names("https://example.com/"))
	require.False(t, names("https://other.example/@bob"))
	require.False(t, names("https://example.com/@bob/outbox"))
	require.False(t, names("https://example.com/@"))
	require.False(t, names("https://example.com/bob"))
	require.False(t, names("/@bob"))
	require.False(t, names("%zz"))

	// With nobody signed in, nothing is compared
	require.False(t, signedInAsSomeoneElse("https://example.com/@bob", "example.com", nil))
}

// FuzzSignedInAsSomeoneElse requires a warning only for a profile path on this host
func FuzzSignedInAsSomeoneElse(f *testing.F) {

	f.Add("https://example.com/@bob")
	f.Add("https://example.com/")
	f.Add("https://other.example/@bob")
	f.Add("https://example.com/@alice/")
	f.Add("%zz")
	f.Add("//example.com/@bob")

	alice := model.NewUser()
	alice.Username = "alice"

	f.Fuzz(func(t *testing.T, me string) {

		if !signedInAsSomeoneElse(me, "example.com", &alice) {
			return
		}

		parsed, err := url.Parse(me)
		require.NoError(t, err)
		require.True(t, strings.EqualFold(parsed.Hostname(), "example.com"), "warned for host %q", parsed.Hostname())
		require.True(t, strings.HasPrefix(parsed.Path, "/@"), "warned for path %q", parsed.Path)
	})
}

// TestOAuthScopeIcons_DrawOnThePage requires every scope icon to draw itself through the icons the page uses
func TestOAuthScopeIcons_DrawOnThePage(t *testing.T) {

	icons := service.NewIcons()

	for _, lookupCode := range append(dataset.OAuthScopes(), describeScopes([]string{"made-up"})...) {
		require.Contains(t, icons.Get(lookupCode.Icon), `bi bi-`+lookupCode.Icon+`"`, "scope %q", lookupCode.Value)
	}
}

// TestConsentPage_DescribesTheRequest renders the consent page and checks what it tells the User
func TestConsentPage_DescribesTheRequest(t *testing.T) {

	client := model.NewOAuthClient()
	client.ClientURL = "https://app.example/"
	client.Name = "app.example"

	user := model.NewUser()
	user.Username = "alice"

	// render draws the page for a request from the URL client, made by the signed-in User
	render := func(t *testing.T, client model.OAuthClient, transaction model.OAuthAuthorizationRequest) string {

		t.Helper()

		request := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=https%3A%2F%2Fapp.example%2F", nil)
		builder := OAuthAuthorization{
			_client:      client,
			_transaction: transaction,
			_user:        &user,
			Theme:        Theme{Common: Common{_factory: newTestDomainFactory(mapof.NewAny(), mapof.NewString()), _request: request}},
		}

		var page strings.Builder
		require.NoError(t, parseConsentPage(t).ExecuteTemplate(&page, "oauth", builder))
		return page.String()
	}

	t.Run("scopes in plain words, in the dataset's order", func(t *testing.T) {

		page := render(t, client, model.OAuthAuthorizationRequest{Scope: "read:statuses made-up profile write:statuses read:lists"})

		// The page lists one row per scope, with no group headers and no warning color
		require.Equal(t, 5, strings.Count(page, `<div class="flex-row margin-top-sm `))
		require.NotContains(t, page, "text-red\">")
		require.NotContains(t, page, ">Read Access<")

		require.Contains(t, page, ">Read Posts<")
		require.Contains(t, page, "See posts and timelines")
		require.Contains(t, page, ">made-up<")
		require.Contains(t, page, `bi bi-person-vcard`)
		require.NotContains(t, page, "Confirm Who You Are")
		require.Less(t, strings.Index(page, "Your Profile"), strings.Index(page, "Read Lists"))
		require.Less(t, strings.Index(page, "Read Lists"), strings.Index(page, "Read Posts"))
	})

	t.Run("no scope asks only who the User is", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{})
		require.Contains(t, page, "Confirm Who You Are")
		require.NotContains(t, page, "Read Access")
	})

	t.Run("data portability keeps its warning", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{Scope: model.OAuthUserTokenScopeActivityPubPortability})
		require.Contains(t, page, `<div class="flex-row margin-top-sm text-red">`)
		require.Contains(t, page, "Export All of Your Data")
		require.Contains(t, page, "only if you are moving your account")
	})

	t.Run("where the client is, and where approval sends the User", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{RedirectURI: "https://app.example/callback"})
		require.Contains(t, page, `App address:</span> https://app.example/`)
		require.Contains(t, page, `Sends you back to:</span> https://app.example/callback`)
	})

	t.Run("out of band sends the User nowhere", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{RedirectURI: model.OAuthRedirectURIOutOfBand})
		require.NotContains(t, page, "Sends you back to:")
	})

	t.Run("a registered client has no address to show", func(t *testing.T) {
		registered := model.NewOAuthClient()
		page := render(t, registered, model.OAuthAuthorizationRequest{RedirectURI: "https://app.example/callback"})
		require.NotContains(t, page, "App address:")
	})

	t.Run("a hint naming someone else warns and offers to switch", func(t *testing.T) {

		page := render(t, client, model.OAuthAuthorizationRequest{Me: "https://example.com/@bob"})
		require.Contains(t, page, "asked to sign in <b>https://example.com/@bob</b>, but you are signed in as <b>@alice@")

		// The switch link signs out, and brings the User back to this request
		match := regexp.MustCompile(`<a hx-post="([^"]*)">Switch Account</a>`).FindStringSubmatch(page)
		require.Len(t, match, 2)
		switchURL, err := url.Parse(html.UnescapeString(match[1]))
		require.NoError(t, err)
		require.Equal(t, "/signout", switchURL.Path)
		require.Equal(t, "/oauth/authorize?client_id=https%3A%2F%2Fapp.example%2F", switchURL.Query().Get("next"))
	})

	t.Run("a hint naming the signed-in User says nothing", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{Me: "https://example.com/@alice"})
		require.NotContains(t, page, "asked to sign in")
	})

	t.Run("the hint is escaped", func(t *testing.T) {
		page := render(t, client, model.OAuthAuthorizationRequest{Me: "https://example.com/@<script>"})
		require.Contains(t, page, "@&lt;script&gt;")
		require.NotContains(t, page, "<script>")
	})
}

// primitiveID returns a fresh ObjectID, as a User other than the test's own would carry
func primitiveID() string {
	return model.NewUser().UserID.Hex()
}
