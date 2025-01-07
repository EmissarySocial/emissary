package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetIntent_Create(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Create"

	// Collect values from the QueryString
	var transaction camper.CreateIntent

	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Error reading form data")
	}

	// Default values here
	onCancel := firstOf(transaction.OnCancel, "/@me")

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Script().Src("/.themes/global/resources/htmx/htmx.min.js").Close()
	b.Close()

	b.Body()

	b.Form("POST", "/@me/intent/create")
	b.Input("hidden", "inReplyTo").Value(transaction.InReplyTo)
	b.Input("hidden", "on-success").Value(transaction.OnSuccess)
	b.Input("hidden", "on-cancel").Value(transaction.OnCancel)

	b.Div().Class("flex-column", "flex-align-stretch", "padding").Style("height:100vh", "max-height:100vh")
	{
		write_intent_header(ctx, b, user)

		b.Textarea("content").Class("flex-grow-1", "margin-vertical", "width-100%").Attr("autofocus", "true").Style("height:100%").InnerHTML(transaction.Content).Close()

		b.Div().Class("flex-shrink-0")
		{
			b.Button().Type("submit").Class("primary").TabIndex("0").InnerText("Create New Post").Close()
			b.A(transaction.OnCancel).Href(onCancel).Class("button").TabIndex("0").InnerText("Cancel")
		}
	}
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

func PostIntent_Create(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Create"

	// Collect values from the Form post
	var transaction camper.CreateIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Error reading form data")
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
	stream.Content = model.NewHTMLContent(transaction.Content)

	// Save the new Stream to the database
	if err := streamService.Save(&stream, "Saved via Activity Intent"); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error saving stream"))
	}

	// Redirect to the "on-success" URL
	return ctx.Redirect(http.StatusSeeOther, onSuccess)
}
