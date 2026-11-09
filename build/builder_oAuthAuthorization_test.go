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
	"github.com/EmissarySocial/emissary/tools/templates"
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

	// The page posts back everything it was shown, with the client named by its own ID
	expected := shown
	expected.ClientID = client.ClientID.Hex()
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
