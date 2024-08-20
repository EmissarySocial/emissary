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
	var txn camper.CreateIntent
	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding form to transaction")
	}

	// Default values here
	onCancel := firstOf(txn.OnCancel, "/@me")

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Close()

	b.Body()

	b.Form("POST", "/@me/intent/create")
	b.Input("hidden", "inReplyTo").Value(txn.InReplyTo)
	b.Input("hidden", "on-success").Value(txn.OnSuccess)
	b.Input("hidden", "on-cancel").Value(txn.OnCancel)

	b.Div().Class("flex-column", "flex-align-stretch", "padding").Style("height:100vh", "max-height:100vh")
	{
		write_intent_header(b, factory.Hostname(), user)

		b.Textarea("content").Class("flex-grow-1", "margin-vertical", "width-100%").Attr("autofocus", "true").Style("height:100%").InnerHTML(txn.Content).Close()

		b.Div().Class("flex-shrink-0")
		{
			b.Button().Type("submit").Class("primary").TabIndex("0").InnerText("Create New Post").Close()
			b.A(txn.OnCancel).Href(onCancel).Class("button").TabIndex("0").InnerText("Cancel")
		}
	}
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

func PostIntent_Create(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Create"

	// Collect values from the Form post
	var txn camper.CreateIntent
	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding form to transaction")
	}

	// Default values here
	onSuccess := firstOf(txn.OnSuccess, "/@me")

	// Create the new Stream
	streamService := factory.Stream()
	stream := model.NewStream()
	stream.TemplateID = firstOf(user.NoteTemplate, "outbox-message")
	stream.ParentID = user.UserID
	stream.ParentIDs = []primitive.ObjectID{user.UserID}
	stream.Label = txn.Name
	stream.Summary = txn.Summary
	stream.Content = model.NewHTMLContent(txn.Content)

	// Save the new Stream to the database
	if err := streamService.Save(&stream, "Saved via Activity Intent"); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error saving stream"))
	}

	// Redirect to the "on-success" URL
	return ctx.Redirect(http.StatusSeeOther, onSuccess)
}
