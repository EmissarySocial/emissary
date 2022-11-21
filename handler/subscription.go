package handler

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetSubscription displays an edit for for a specific subscription
func GetSubscription(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetSubscription"

	return func(ctx echo.Context) error {

		// Load all pre-requisites
		factory, subscription, userID, subscriptionID, err := subscription_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading subscription", userID, subscriptionID)
		}

		// Create a new form
		folderService := factory.Folder()
		folders := subscription_folderOptions(folderService, userID)
		form := subscription_getForm(folders)

		if subscriptionID == "new" {
			form.Element.Label = "Add a Subscription"
		} else {
			form.Element.Label = "Edit Subscription"
		}

		html, err := form.Editor(subscription, nil)

		if err != nil {
			return derp.Wrap(err, location, "Error creating form editor", nil)
		}

		// Wrap the form as a modal dialog (with submit buttons)
		html = render.WrapModalForm(ctx.Response(), "/@me/pub/subscriptions/"+subscriptionID, html)

		// Done.
		return ctx.HTML(http.StatusOK, html)
	}
}

func PostSubscription(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostSubscription"

	return func(ctx echo.Context) error {

		var transaction struct {
			URL           string             `form:"url"`
			FolderID      primitive.ObjectID `form:"folderId"`
			PollDuration  int                `form:"pollDuration"`
			PurgeDuration int                `form:"purgeDuration"`
		}

		// Load all pre-requisites
		factory, subscription, userID, subscriptionID, err := subscription_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading subscription", userID, subscriptionID)
		}

		// Create a new form
		folderService := factory.Folder()
		folders := subscription_folderOptions(folderService, userID)
		form := subscription_getForm(folders)

		// Collect data from the form POST

		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error reading form data", nil)
		}

		if err := form.Schema.Validate(transaction); err != nil {
			return derp.Wrap(err, location, "Subscription Data is invalid", transaction)
		}

		subscription.URL = transaction.URL
		subscription.FolderID = transaction.FolderID
		subscription.PollDuration = transaction.PollDuration
		subscription.PurgeDuration = transaction.PurgeDuration

		// Save the subscription to the database
		if err := factory.Subscription().Save(&subscription, "Updated by User"); err != nil {
			return derp.Wrap(err, location, "Error saving subscription", subscription)
		}

		// Close the Modal Dialog and return
		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)

	}
}

func GetDeleteSubscription(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.DeleteSubscription"

	return func(ctx echo.Context) error {

		_, subscription, userID, subscriptionID, err := subscription_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading subscription", userID, subscriptionID)
		}

		b := html.New()

		b.H2().InnerHTML("Delete This Subscription?").Close()
		b.Div().Class("space-below").InnerHTML(subscription.Label).Close()
		b.Div().Class("space-below").InnerHTML(subscription.URL).Close()

		b.Button().Class("warning").
			Attr("hx-post", "/@me/pub/subscriptions/"+subscription.SubscriptionID.Hex()+"/delete").
			Attr("hx-swap", "none").
			InnerHTML("Delete Subscription").
			Close()

		b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()
		b.CloseAll()

		result := render.WrapModal(ctx.Response(), b.String())
		io.WriteString(ctx.Response(), result)
		return nil
	}
}

func PostDeleteSubscription(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.DeleteSubscription"

	return func(ctx echo.Context) error {

		factory, subscription, userID, subscriptionID, err := subscription_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading subscription", userID, subscriptionID)
		}

		// Delete the subscription
		if err := factory.Subscription().Delete(&subscription, "Deleted by User"); err != nil {
			return derp.Wrap(err, location, "Error deleting subscription", subscription)
		}

		// Close the Modal Dialog and return
		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

// subscription_common is a helper method that loads all of the standard pre-requisites for subscription handlers
func subscription_common(serverFactory *server.Factory, ctx echo.Context) (*domain.Factory, model.Subscription, primitive.ObjectID, string, error) {

	const location = "handler.subscriptionLoad"

	// Get the factory for this domain
	factory, err := serverFactory.ByContext(ctx)

	if err != nil {
		return nil, model.Subscription{}, primitive.NilObjectID, "", derp.Wrap(err, location, "Error getting server factory")
	}

	// Validate the user's session
	sterankoContext := ctx.(*steranko.Context)
	authorization := getAuthorization(sterankoContext)

	// Requre that users are signed in to use this modal
	if !authorization.IsAuthenticated() {
		return nil, model.Subscription{}, primitive.NilObjectID, "", derp.NewUnauthorizedError(location, "User is not authenticated", nil)
	}

	// Create/Load the subscription
	subscriptionService := factory.Subscription()
	subscription := model.NewSubscription()
	subscriptionID := ctx.Param("subscription")
	userID := authorization.UserID

	if err := subscriptionService.LoadByToken(userID, subscriptionID, &subscription); err != nil {
		return nil, model.Subscription{}, primitive.NilObjectID, "", derp.Wrap(err, location, "Error loading subscription", subscriptionID)
	}

	return factory, subscription, userID, subscriptionID, nil
}

// subscription_getForm returns a form for adding/editing subscriptions
func subscription_getForm(folders []form.LookupCode) form.Form {

	folderIDs := slice.Map(folders, func(folder form.LookupCode) string { return folder.Value })

	return form.Form{
		Schema: schema.New(schema.Object{
			Properties: schema.ElementMap{
				"url":           schema.String{MaxLength: 512, Required: true},
				"folderId":      schema.String{Format: "objectId", Enum: folderIDs, Required: true},
				"pollDuration":  schema.Integer{Default: null.NewInt64(24), Minimum: null.NewInt64(1), Maximum: null.NewInt64(24 * 30), Required: true},
				"purgeDuration": schema.Integer{Default: null.NewInt64(14), Minimum: null.NewInt64(1), Maximum: null.NewInt64(365), Required: true},
			},
		}),
		Element: form.Element{
			Type: "layout-vertical",
			Children: []form.Element{
				{
					Type:        "text",
					Label:       "Website URL",
					Path:        "url",
					Description: "Enter the URL of the website you want to subscribe to.",
				},
				{
					Type:  "select",
					Label: "Folder",
					Path:  "folderId",
					Options: maps.Map{
						"enum": folders,
					},
					Description: "Automatically add items to this folder.",
				},
				{
					Type:        "select",
					Label:       "Poll Frequency",
					Description: "How often should this site be checked for new articles?",
					Path:        "pollDuration",
					Options: maps.Map{
						"enum": []form.LookupCode{
							{Value: "1", Label: "Hourly"},
							{Value: "6", Label: "Every 6 Hours"},
							{Value: "12", Label: "Every 12 Hours"},
							{Value: "24", Label: "Once per Day"},
							{Value: "168", Label: "Once per Week"},
							{Value: "720", Label: "Once per Month"},
						},
					},
				},
				{
					Type:        "select",
					Label:       "Remove After",
					Description: "Read items will be automatically deleted after this amount of time.",
					Path:        "purgeDuration",
					Options: maps.Map{
						"enum": []form.LookupCode{
							{Value: "1", Label: "1 Day"},
							{Value: "7", Label: "1 Week"},
							{Value: "14", Label: "2 Weeks"},
							{Value: "30", Label: "1 Month"},
							{Value: "60", Label: "2 Months"},
							{Value: "90", Label: "3 Months"},
							{Value: "180", Label: "6 Months"},
							{Value: "365", Label: "1 Year"},
						},
					},
				},
			},
		},
	}
}

// subscription_folderOptions returns an array of form.LookupCodes that represents all of the folders
// that belong to the currently logged in user.
func subscription_folderOptions(folderService *service.Folder, authenticatedID primitive.ObjectID) []form.LookupCode {

	folders, _ := folderService.QueryByUserID(authenticatedID)
	result := make([]form.LookupCode, len(folders))

	for index, folder := range folders {
		result[index] = folder.LookupCode()
	}

	return result
}
