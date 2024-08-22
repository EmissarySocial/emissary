package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
)

func GetIntent_Follow(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Follow"

	// Collect values from the QueryString
	var txn camper.FollowIntent
	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding form to transaction")
	}

	// Default values here
	onCancel := firstOf(txn.OnCancel, "/@me")

	// Try to load the remote Actor to be followed
	activityService := factory.ActivityStream()
	actor, err := activityService.Load(txn.Object, sherlock.AsActor())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load object", txn)
	}

	// Try to load an existing "Following" record (allow "NOT FOUND" errors)
	followingService := factory.Following()
	following := model.NewFollowing()
	if err := followingService.LoadByURL(user.UserID, actor.ID(), &following); err != nil {
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading existing following")
		}
	}

	// Generate the input form as HTML
	lookupProvider := factory.LookupProvider(user.UserID)
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
	b.Input("hidden", "url").Value(txn.Object)
	b.Input("hidden", "on-success").Value(txn.OnSuccess)

	b.Div().Class("padding", "flex-column").Style("height:100vh", "max-height:100vh")
	{
		write_intent_header(ctx, b, user)

		b.Div().Class("card", "flex-grow", "padding")
		{
			b.Div().Class("flex-row", "flex-align-center", "margin-bottom")
			{
				b.Img(actor.Icon().Href()).Class("circle-48", "flex-shrink-0").Close()
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
			b.A(txn.OnCancel).Href(onCancel).Class("button").TabIndex("0").InnerText("Cancel")
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

func PostIntent_Follow(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Follow"

	// Collect values from the Form post
	txn := mapof.NewAny()

	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding form to transaction")
	}

	// Default values here
	onSuccess := firstOf(txn.GetString("on-success"), "/@me")

	// Follow the new Stream
	followingService := factory.Following()
	following := model.NewFollowing()
	following.UserID = user.UserID

	// Update the Following with values from the user
	form := getForm_FollowingIntent()
	if err := form.SetAll(&following, txn, factory.LookupProvider(user.UserID)); err != nil {
		return derp.Wrap(err, location, "Error setting form values")
	}

	// Save the new Stream to the database
	if err := followingService.Save(&following, "Created via Activity Intent"); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error saving stream"))
	}

	// Redirect to the "on-success" URL
	return ctx.Redirect(http.StatusSeeOther, onSuccess)
}
