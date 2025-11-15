package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetIntent_Follow(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Follow"

	// Collect values from the QueryString
	var transaction camper.FollowIntent
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Unable to read form data")
	}

	// Default values here
	onCancel := firstOf(transaction.OnCancel, "/@me")

	// Try to load the remote Actor to be followed
	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)
	actor, err := activityService.Client().Load(transaction.Object, sherlock.AsActor())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load object", transaction)
	}

	// Try to load an existing "Following" record (allow "NOT FOUND" errors)
	followingService := factory.Following()
	following := model.NewFollowing()
	if err := followingService.LoadByURL(session, user.UserID, actor.ID(), &following); err != nil {
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Unable to load existing following")
		}
	}

	// Generate the input form as HTML
	lookupProvider := factory.LookupProvider(ctx.Request(), session, user.UserID)
	formStruct := getForm_FollowingIntent()
	formHTML, err := formStruct.Editor(following, lookupProvider)

	if err != nil {
		return derp.Wrap(err, location, "Error building form")
	}

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Script().Src("/.themes/global/resources/htmx/htmx.min.js").Close()
	b.Close()

	b.Body()

	b.Form("POST", "/@me/intent/follow")
	b.Input("hidden", "url").Value(transaction.Object)
	b.Input("hidden", "on-success").Value(transaction.OnSuccess)

	b.Div().Class("padding", "flex-column").Style("height:100vh", "max-height:100vh")
	{
		write_intent_header(ctx, b, user)

		b.Div().Class("card", "flex-grow", "padding")
		{
			b.Div().Class("flex-row", "flex-align-center", "margin-bottom")
			{
				b.Img(actor.Icon().Href()).Class("circle width-48", "flex-shrink-0").Close()
				b.Div().Class("flex-grow")
				{
					b.Div().Class("text-lg", "bold", "margin-none").InnerText("Follow " + actor.Name())
					b.Div().Class("text-gray")
					b.A(actor.URL()).InnerText(ActorUsername(actor))
					b.Close()
				}
				b.Close()
			}
			b.Close()

			b.Div().Class("flex-grow-1").EndBracket()
			b.WriteString(formHTML)
			b.Close()
		}
		b.Close()

		b.Div().Class("margin-top")
		{
			b.Button().Type("submit").Class("primary").TabIndex("0").InnerText("Follow " + actor.Name()).Close()
			b.A("/@me/intent/continue?url=" + url.QueryEscape(onCancel)).Class("button").TabIndex("0").InnerText("Cancel")
		}
	}
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

// getForm_FollowingIntent returns the form object to display/update the Following Intent
func getForm_FollowingIntent() form.Form {

	return form.Form{
		Schema: schema.New(model.FollowingSchema()),
		Element: form.Element{
			Type: "layout-vertical",
			Children: []form.Element{
				{
					Type: "hidden",
					Path: "url",
				},
				{
					Type:        "select",
					Label:       "Inbox Folder",
					Path:        "folderId",
					Description: "Where should messages from this source be placed?",
					Options:     mapof.Any{"provider": "folders"},
				},
				{
					Type:        "select",
					Label:       "Message Types",
					Path:        "behavior",
					Description: "What kinds of posts should be shown in my timeline?",
					Options:     mapof.Any{"provider": "following-behaviors"},
				},
				{
					Type:        "select",
					Label:       "Shared Blocks",
					Path:        "ruleAction",
					Description: "How should blocks from this source be handled?",
					Options:     mapof.Any{"provider": "following-rule-actions"},
				},
				{
					Type: "toggle",
					Path: "collapseThreads",
					Options: mapof.Any{
						"true-text":  "Group messages into a single thread",
						"false-text": "Show all messages separately",
					},
				},
				{
					Type: "toggle",
					Path: "isPublic",
					Options: mapof.Any{
						"true-text":  "Public: This 'Follow' is visible on my profile",
						"false-text": "Private: This 'Follow' is hidden from others",
					},
				},
			},
		},
	}
}

func PostIntent_Follow(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetIntent_Follow"

	// Collect values from the Form post
	transaction, err := formdata.Parse(ctx.Request())

	if err != nil {
		return derp.Wrap(err, location, "Unable to read form data")
	}

	// Default values here
	onSuccess := firstOf(transaction.Get("on-success"), "/@me")

	// Follow the new Stream
	followingService := factory.Following()
	following := model.NewFollowing()
	following.UserID = user.UserID

	// Update the Following with values from the user
	form := getForm_FollowingIntent()
	if err := form.SetURLValues(&following, transaction, factory.LookupProvider(ctx.Request(), session, user.UserID)); err != nil {
		return derp.Wrap(err, location, "Error setting form values")
	}

	// Save the new Stream to the database
	if err := followingService.Save(session, &following, "Created via Activity Intent"); err != nil {
		return derp.Wrap(err, location, "Unable to save stream")
	}

	// Return the "on-success" response
	return ctx.HTML(http.StatusOK, getIntent_Continue(onSuccess))
}
