package handler

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/uri"
	"github.com/stretchr/testify/require"
)

// GetOutboundIntent forwards a visitor to a URL TEMPLATE published by their home server, and the
// account in the request decides which server is asked — so the template is remote, caller-steered
// input. These tests pin the composition the handler performs, populate and then validate, over the
// shapes that reached ctx.Redirect before the guard was added.
//
// They do NOT drive the handler, which needs a configured Factory and network access before it can
// resolve any template, so they cannot catch the guard itself being deleted.

// intentValues are the substitution values an Activity Intent carries.
func intentValues() url.Values {
	return url.Values{"content": {"hello"}, "object": {"https://remote.example/post/1"}}
}

// A hostile or compromised home server must not be able to choose the destination.
func TestOutboundIntent_RejectsHostileTemplates(t *testing.T) {

	client := camper.New()

	for _, template := range []string{
		"/signout?{content}",                                // a route on THIS server
		"/@me/settings/delete?{content}",                    // ...including a destructive one
		"//phish.example/{content}",                         // protocol-relative reads as off-site
		`/\phish.example/{content}`,                         // backslash opens an authority
		`///phish.example/{content}`,                        // extra slashes do too
		"javascript:alert(document.domain)//{content}",      // script scheme
		"data:text/html,<script>alert(1)</script>{content}", // data scheme
	} {
		nextURL := client.PopulateTemplate(template, intentValues())
		require.False(t, uri.IsValidRedirectURL(nextURL),
			"template %q populated to %q, which must never be a destination", template, nextURL)
	}
}

// A real home server's template must still work, or every Activity Intent breaks.
func TestOutboundIntent_AcceptsRealHomeServers(t *testing.T) {

	client := camper.New()

	for _, template := range []string{
		"https://mastodon.social/share?text={content}",
		"https://example.com/@me/intent/create?content={content}&inReplyTo={object}",
		"http://localhost:8080/share?text={content}", // local development home server
	} {
		nextURL := client.PopulateTemplate(template, intentValues())
		require.True(t, uri.IsValidRedirectURL(nextURL),
			"template %q populated to %q, which must be a valid destination", template, nextURL)
	}
}

// A hostile VALUE cannot move the destination either: PopulateTemplate query-escapes everything it
// substitutes, so an injected host stays inside the query string where it cannot steer the browser.
func TestOutboundIntent_HostileValuesStayInTheQuery(t *testing.T) {

	client := camper.New()

	hostile := url.Values{"content": {"https://evil.example/?x="}, "object": {"//evil.example/#"}}
	nextURL := client.PopulateTemplate("https://home.example.com/share?text={content}&o={object}", hostile)

	require.True(t, uri.IsValidRedirectURL(nextURL))
	require.Equal(t, "home.example.com", uri.Hostname(nextURL))
}
