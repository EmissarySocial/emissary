package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetIntent_Create renders the Activity Intent form for composing a new post
func GetIntent_Create(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Create"

	// Collect values from the QueryString
	var transaction camper.CreateIntent

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Reading form data")
	}

	// Default values here
	onCancel := firstOf(transaction.OnCancel, "/@me") // nolint:scopeguard

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Script().Src("/.themes/global/resources/htmx-1.9.12/htmx.min.js").Close()
	b.Close()

	b.Body()

	b.Form("POST", "/@me/intent/create")
	b.Input("hidden", "inReplyTo").Value(transaction.InReplyTo)
	b.Input("hidden", "on-success").Value(transaction.OnSuccess)
	b.Input("hidden", "on-cancel").Value(transaction.OnCancel)

	b.Div().Class("flex-column", "flex-align-stretch", "padding").Style("height:100vh", "max-height:100vh")
	{
		write_intent_header(ctx, b, user)

		// SECURITY: `content` is a caller-supplied query param. Use InnerText (which HTML-escapes)
		// and NOT InnerHTML — otherwise a `</textarea>`-prefixed value breaks out of the textarea
		// and executes as reflected XSS. A textarea is a plain-text field, so escaping is correct.
		b.Textarea("content").Class("flex-grow-1", "margin-vertical", "width-100%").Attr("autofocus", "true").Style("height:100%").InnerText(transaction.Content).Close()

		b.Div().Class("flex-shrink-0")
		{
			b.Button().Type("submit").Class("primary").TabIndex("0").InnerText("Create New Post").Close()
			b.A("/@me/intent/continue?url=" + url.QueryEscape(onCancel)).Class("button").TabIndex("0").InnerText("Cancel")
		}
	}
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

// PostIntent_Create publishes the post composed on the Create Activity Intent form
func PostIntent_Create(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Create"

	// Collect values from the Form post
	var transaction camper.CreateIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Reading form data")
	}

	// Default values here
	onSuccess := firstOf(transaction.OnSuccess, "/@me")

	// Create the new Stream
	streamService := factory.Stream()
	stream := model.NewStream()
	stream.TemplateID = firstOf(user.NoteTemplate, "outbox-message")
	stream.ParentID = user.UserID
	stream.ParentIDs = []primitive.ObjectID{user.UserID}
	stream.Label = transaction.Name
	stream.Summary = transaction.Summary
	stream.InReplyTo = transaction.InReplyTo
	stream.Content = model.NewHTMLContent(transaction.Content)

	// Save the new Stream to the database
	if err := streamService.Publish(session, user, &stream, "published", true, false); err != nil {
		return derp.Wrap(err, location, "Publishing stream")
	}

	// Return the "on-success" response
	return ctx.HTML(http.StatusOK, getIntent_Continue(onSuccess))
}
