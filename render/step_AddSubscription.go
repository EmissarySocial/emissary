package render

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepAddSubscription is an action that can add a subscription for the current user.
type StepAddSubscription struct {
}

func (step StepAddSubscription) Get(renderer Renderer, buffer io.Writer) error {

	// Requre that users are signed in to use this modal
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError("render.StepAddSubscription", "User is not authenticated", nil)
	}

	// Get the request context
	context := renderer.context()

	// Create a new form
	form := step.getForm(renderer)
	html, err := form.Editor(nil, nil)

	if err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error creating form editor", nil)
	}

	// Wrap the form as a modal dialog (with submit buttons)
	html = WrapModalForm(context.Response(), renderer.URL(), html)

	// Done.
	return context.HTML(http.StatusOK, html)
}

func (step StepAddSubscription) UseGlobalWrapper() bool {
	return false
}

func (step StepAddSubscription) Post(renderer Renderer) error {

	var transaction subscription_transaction

	// Guarantee that the user is signed in.
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError("render.StepAddSubscription", "User is not authenticated", nil)
	}

	// Collect data from the form POST
	context := renderer.context()

	if err := context.Bind(&transaction); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error reading form data", nil)
	}

	if err := step.getForm(renderer).Schema.Validate(transaction); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Subscription Data is invalid", transaction)
	}

	// Populate a new subscription object with the form data
	subscription := model.NewSubscription()
	subscription.UserID = renderer.AuthenticatedID()
	subscription.Method = model.SubscriptionMethodRSS
	subscription.URL = transaction.URL
	subscription.PollDuration = transaction.PollDuration
	subscription.PurgeDuration = transaction.PurgeDuration
	subscription.FolderID = transaction.FolderID

	// Save the subscription to the database
	factory := renderer.factory()
	subscriptionService := factory.Subscription()

	if err := subscriptionService.Save(&subscription, "Created"); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error saving subscription", subscription)
	}

	// Close the Modal Dialog and return
	CloseModal(context, "")
	return context.NoContent(http.StatusOK)
}

func (step StepAddSubscription) getForm(renderer Renderer) form.Form {
	result := subscription_getForm(renderer)
	result.Element.Label = "Add a New Subscription"
	return result
}

type subscription_transaction struct {
	URL           string             `form:"url"           path:"url"`
	FolderID      primitive.ObjectID `form:"folderId"      path:"folderId"`
	PollDuration  int                `form:"pollDuration"  path:"pollDuration"`
	PurgeDuration int                `form:"purgeDuration" path:"purgeDuration"`
}

func subscription_getForm(renderer Renderer) form.Form {
	return form.Form{
		Schema: schema.New(schema.Object{
			Properties: schema.ElementMap{
				"url":           schema.String{MaxLength: 512, Required: true},
				"folderId":      schema.String{Format: "objectId", Required: true},
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
						"enum": subscription_folderOptions(renderer.factory().Folder(), renderer.AuthenticatedID()),
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

	folders, err := folderService.QueryByUserID(authenticatedID)

	if err != nil {
		return make([]form.LookupCode, 0)
	}

	result := make([]form.LookupCode, len(folders))

	for index, folder := range folders {
		result[index] = form.LookupCode{
			Value: folder.FolderID.Hex(),
			Label: folder.Label,
		}
	}

	return result
}
