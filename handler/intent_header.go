package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/html"
)

func write_intent_header(b *html.Builder, hostname string, user *model.User) {

	b.Div().Class("flex-shrink-0", "flex-row", "flex-align-stretch", "margin-bottom", "text-sm")
	{
		b.Img(user.ActivityPubIconURL()).Class("circle-48", "flex-shrink-0").Close()
		b.Div().Class("flex-grow")
		{
			b.Div().Class("text-gray").InnerText("Signed In As:")
			b.A(user.ProfileURL).Attr("target", "_blank").Class("bold", "text-plain").InnerText("@" + user.Username + "@" + hostname)
		}
		b.Close()
		// b.Button().Class("text-xs", "button").Data("hx-post", "/signout").Data("hx-swap", "none").Script("").InnerText("Sign Out").Close()
	}
	b.Close()

}
