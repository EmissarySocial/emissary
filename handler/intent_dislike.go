package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/html"
	"github.com/benpate/icon"
	"github.com/benpate/steranko"
)

// GetIntent_Dislike renders the Activity Intent confirmation page for disliking a remote object
func GetIntent_Dislike(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Dislike"

	// Collect values from the QueryString
	var transaction camper.DislikeIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Reading form data")
	}

	// Load the object to be disliked
	client := factory.ActivityStream().AppClient()
	object, err := client.Load(transaction.Object)

	if err != nil {
		return derp.Wrap(err, location, "Loading object", ctx.Request().URL.String(), ctx.Request().URL, transaction)
	}

	// Ask the User to confirm
	return ctx.HTML(http.StatusOK, getIntent_Dislike(ctx, user, factory.Icons(), transaction, object))
}

// getIntent_Dislike returns the confirmation page, whose form posts the object to PostIntent_Dislike
func getIntent_Dislike(ctx *steranko.Context, user *model.User, icons icon.Provider, transaction camper.DislikeIntent, object streams.Document) string {

	// Build the page's head
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/resources/bootstrap-icons-1.13.1/bootstrap-icons.min.css").Close()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Script().Src("/.themes/global/resources/htmx-1.9.12/htmx.min.js").Close()
	b.Close()

	b.Body().Style("overflow-y:hidden")

	// RULE: The form carries the object itself, because its action has no query string to carry it
	b.Form("POST", "/@me/intent/dislike")
	b.Input("hidden", "object").Value(transaction.Object)
	b.Input("hidden", "on-success").Value(transaction.OnSuccess)
	b.Input("hidden", "on-cancel").Value(transaction.OnCancel)

	b.Div().Class("flex-column", "padding").Style("height:99vh", "max-height:99vh")
	{
		write_intent_header(ctx, b, user)

		// Show the object being disliked
		b.Div().Class("flex-column", "flex-grow-1", "card", "padding").Style("overflow-y:scroll")
		{
			if name := object.Name(); name != "" {
				b.Div().Class("margin-top-none", "text-lg", "bold").InnerText(name).Close()
			}

			if attributedTo := object.AttributedTo(); attributedTo.NotNil() {

				b.Div().Class("flex-row", "margin-bottom")
				{
					b.Img(safeIntentURL(attributedTo.Icon().Href())).Class("flex-shrink-0", "circle", "width-32").Close()
					b.Div().Class("text-sm", "margin-none")
					{
						b.Div().Class("bold").InnerText(attributedTo.Name()).Close()
						b.Div().Class("text-gray").InnerText(ActorUsername(attributedTo)).Close()
					}
					b.Close()
				}
				b.Close()
			}

			// SECURITY: object is loaded from the caller-supplied `object` URL, so its summary/content
			// are untrusted remote HTML. write_intentObjectContent sanitizes before InnerHTML.
			write_intentObjectContent(b, object.Summary(), object.Content())
		}
		b.Close()

		// Confirm, or cancel
		b.Div().Class("margin-top")
		{
			onCancel := firstOf(transaction.OnCancel, "/@me")
			b.Button().Type("submit").Class("primary").InnerHTML(icons.Get("thumbs-down-fill") + " Dislike This").Close()
			b.A("/@me/intent/continue?url=" + url.QueryEscape(onCancel)).Class("button").TabIndex("0").InnerText("Cancel")
		}
		b.Close()
	}
	b.CloseAll()

	// Thumbs down, chin up
	return b.String()
}

// PostIntent_Dislike records a Dislike from the Activity Intent form
func PostIntent_Dislike(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return postIntent_Response(ctx, factory, session, user, vocab.ActivityTypeDislike)
}
