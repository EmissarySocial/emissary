package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/html"
)

// intent_Header writes the standard HTML header for an Intent page
func intent_header(user *model.User, b *html.Builder) {

	b.Div().Class("flex-shrink-0", "flex-row", "margin-bottom")
	{
		b.Img(user.ActivityPubIconURL()).Class("circle-48", "flex-shrink-0").Close()
		b.Div().Class("flex-grow")
		{
			b.Div().Class("bold", "text-lg", "margin-vertical-none").InnerText(user.DisplayName)
			b.Div().Class("text-sm", "text-gray").InnerText("@" + user.Username)
		}
		b.Close()
	}
	b.Close()
}
