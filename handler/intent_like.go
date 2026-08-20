package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
)

// GetIntent_Like renders the Activity Intent confirmation page for liking a remote object
func GetIntent_Like(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Like"

	// Collect values from the QueryString
	var transaction camper.LikeIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Reading form data")
	}

	// Default values here
	onCancel := firstOf(transaction.OnCancel, "/@me") // nolint:scopeguard

	client := factory.ActivityStream().AppClient()
	object, err := client.Load(transaction.Object)

	if err != nil {
		return derp.Wrap(err, location, "Loading object", ctx.Request().URL.String(), ctx.Request().URL, transaction)
	}

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/resources/bootstrap-icons-1.13.1/bootstrap-icons.min.css").Close()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Script().Src("/.themes/global/resources/htmx-1.9.12/htmx.min.js").Close()
	b.Close()

	b.Body().Style("overflow-y:hidden")

	b.Form("POST", "/@me/intent/like")
	b.Input("hidden", "on-success").Value(transaction.OnSuccess)
	b.Input("hidden", "on-cancel").Value(transaction.OnCancel)

	b.Div().Class("flex-column", "padding").Style("height:99vh", "max-height:99vh")
	{
		write_intent_header(ctx, b, user)

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

		b.Div().Class("margin-top")
		{
			icons := factory.Icons()
			b.Button().Type("submit").Class("primary").InnerHTML(icons.Get("thumbs-up-fill") + " Like This").Close()
			b.A("/@me/intent/continue?url=" + url.QueryEscape(onCancel)).Class("button").TabIndex("0").InnerText("Cancel")
		}
	}
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

// PostIntent_Like records a Like from the Activity Intent form
func PostIntent_Like(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return postIntent_Response(ctx, factory, session, user, vocab.ActivityTypeLike)
}

// postIntent_Response records a Like or Dislike from an Activity Intent form, idempotently
func postIntent_Response(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User, responseType string) error {

	const location = "handler.postIntent_Response"

	// Collect values from the Form post
	var transaction camper.LikeIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Reading form data")
	}

	// Default values here
	onSuccess := firstOf(transaction.OnSuccess, "/@me")

	// Save the Response via SetResponse, which publishes the activity, keeps Likes and Dislikes
	// mutually exclusive, and makes a repeated intent idempotent.  This form is posted from
	// another website and never reads back the reaction it is setting, so a resubmit must
	// confirm the reaction rather than toggle it back off.
	responseService := factory.Response()

	if err := responseService.SetResponse(session, user, transaction.Object, responseType, ""); err != nil {
		return derp.Wrap(err, location, "Saving response", transaction)
	}

	// Return the "on-success" response
	return ctx.HTML(http.StatusOK, getIntent_Continue(onSuccess))
}
