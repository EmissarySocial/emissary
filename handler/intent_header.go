package handler

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/domain"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
)

func write_intent_header(ctx *steranko.Context, b *html.Builder, user *model.User) {

	currentURL := ctx.Request().URL.String()
	hostname := domain.NameOnly(ctx.Request().Host)

	b.Div().Class("flex-shrink-0", "flex-row", "flex-align-stretch", "margin-bottom")
	{
		b.Img(user.ActivityPubIconURL()).Class("circle-32", "flex-shrink-0").Close()
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
