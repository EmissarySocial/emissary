package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// GetIntent_Continue renders the "return to the calling site" page shown after an Activity Intent completes
func GetIntent_Continue(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	url := first.String(ctx.QueryParam("url"), "/@me")
	return ctx.HTML(http.StatusOK, getIntent_Continue(url))
}

// getIntent_Continue renders the continue page for a URL, neutralizing unsafe schemes before they reach the href
func getIntent_Continue(url string) string {

	// (close) directive can be handled without a confirmation page
	if url == "(close)" {
		return "<script>window.close();</script>"
	}

	// Neutralize dangerous targets before they reach the link href. The "url"
	// value ultimately comes from a caller-supplied Activity Intent (`on-success`
	// / `on-cancel`), so a scheme like "javascript:" would execute on click.
	// Unsafe values fall back to the user's home page. See uri.IsSafeRedirectURL.
	if !uri.IsSafeRedirectURL(url) {
		url = "/@me"
	}

	// Otherwise, prevent open redirect attacks by
	// displaying a confirmation page that shows the next URL to the user
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Close()

	b.Body()
	b.Div().ID("main").Style("display:none")
	b.H1().InnerText("Returning to Your Work").Close()
	b.Div().InnerText("Click here to return to your previous workflow").Close()
	b.Div().Class("bold").InnerText(url).Close()
	b.Div()
	b.A(url).InnerText("Continue &rarr;").Close()

	return b.String()
}
