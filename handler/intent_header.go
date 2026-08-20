package handler

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// safeIntentURL returns rawURL when it is safe to place in a link href or image src, and an empty
// string otherwise. Intent pages build hrefs/srcs from caller-supplied or remote-fetched values
// (e.g. a fetched actor's URL/icon), so a "javascript:"/"data:"/protocol-relative value must never
// reach the DOM. Returning "" makes the html builder omit the attribute entirely (a harmless,
// non-navigating element) rather than emit a dangerous one. See uri.IsSafeRedirectURL.
func safeIntentURL(rawURL string) string {

	if uri.IsSafeRedirectURL(rawURL) {
		return rawURL
	}

	return ""
}

// write_intentObjectContent renders the summary (preferred) or content of a remote object into the
// builder. The object is loaded from a caller-supplied URL, so its summary/content are UNTRUSTED
// remote HTML and MUST be sanitized before InnerHTML — otherwise an attacker who hosts the object
// can inject script into the Emissary origin. Sanitizing (not escaping) keeps legitimate post
// markup intact while stripping scripts and event handlers.
func write_intentObjectContent(b *html.Builder, summary string, content string) {

	if summary != "" {
		b.Div().Class("flex-grow-1").InnerHTML(convert.SanitizeHTML(summary)).Close()
		return
	}

	if content != "" {
		b.Div().Class("flex-grow-1").InnerHTML(convert.SanitizeHTML(content)).Close()
	}
}

// write_intent_header renders the "signed in as" banner shown at the top of every Activity Intent page
func write_intent_header(ctx *steranko.Context, b *html.Builder, user *model.User) {

	currentURL := ctx.Request().URL.String()
	hostname := uri.Hostname(user.ProfileURL)

	b.Div().Class("flex-shrink-0", "flex-row", "flex-align-stretch", "margin-bottom")
	{
		b.Div().Class("width-32")
		b.Img(user.ActivityPubIconURL()).Class("circle width-32", "flex-shrink-0").Close()
		b.Close()
		b.Div().Class("flex-grow")
		{
			b.Div().Class("text-xs", "text-gray", "margin-none").InnerText("Signed In As:")
			b.A(user.ProfileURL).Attr("target", "_blank").Class("bold", "text-plain", "text-sm").InnerText("@" + user.Username + "@" + hostname)
		}
		b.Close()
		b.Span().
			Class("button", "text-sm").
			Data("hx-post", "/signout?next="+url.QueryEscape(currentURL)).
			Data("hx-swap", "none").
			InnerText("Sign Out").
			Close()
	}
	b.Close()
}
