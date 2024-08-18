package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetIntentInfo(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetIntentInfo"

	return func(ctx echo.Context) error {

		// Collect intentType
		intentType := ctx.QueryParam("intent")

		if intentType == "" {
			return derp.NewBadRequestError(location, "You must specify an intent")
		}

		// Collect accountID
		accountID := ctx.QueryParam("account")

		if accountID == "" {
			return derp.NewBadRequestError(location, "You must specify a Fediverse account")
		}

		// Get the domain factory
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory by context")
		}

		// Look up the account via the ActivityService
		activityService := factory.ActivityStream()
		actor, err := activityService.Load(accountID, sherlock.AsActor())

		if err != nil {
			return derp.Wrap(err, location, "Error loading account from ActivityService")
		}

		// Return the account information to the client
		ctx.Response().Header().Set("Hx-Push-Url", "false")

		return ctx.JSON(http.StatusOK, mapof.Any{
			vocab.PropertyID:   actor.ID(),
			vocab.PropertyName: actor.Name(),
			vocab.PropertyIcon: actor.Icon().Href(),
			vocab.PropertyURL:  actor.URL(),
		})
	}
}

func GetIntent_Create(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Create"

	txn := camper.NewCreateIntent()
	onCancel := firstOf(txn.OnCancel, "/@me")

	// Collect values from the QueryString
	if err := ctx.Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding form to transaction")
	}

	// Buiild HTML response
	b := html.New()

	b.HTML()
	b.Head()
	b.Link("stylesheet", "/.themes/global/stylesheet").Close()
	b.Link("stylesheet", "/.themes/default/stylesheet").Close()
	b.Close()

	b.Body()
	b.Div().Class("padding")
	b.Form("POST", "/@me/intent/create")
	b.Input("hidden", "inReplyTo").Value(txn.InReplyTo)
	b.Input("hidden", "on-success").Value(txn.OnSuccess)
	b.Input("hidden", "on-cancel").Value(txn.OnCancel)

	b.Div().Class("margin-vertical")
	b.Textarea("content").Class("width-100%").Attr("rows", "8").InnerHTML(txn.Content).Close()
	b.Close()

	b.Div()
	b.Button().Type("submit").Class("primary").InnerText("Create New Post").Close()
	b.A(txn.OnCancel).Href(onCancel).Class("button").InnerText("Cancel")

	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

func PostIntent_Create(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetIntent_Create"

	txn := camper.NewCreateIntent()
	onSuccess := firstOf(txn.OnSuccess, "/@me")

	// Create the new Stream
	streamService := factory.Stream()
	stream := model.NewStream()
	stream.TemplateID = firstOf(user.NoteTemplate, "stream-outbox-message")
	stream.ParentID = user.UserID
	stream.ParentIDs = []primitive.ObjectID{user.UserID}
	stream.Label = txn.Name
	stream.Summary = txn.Summary
	stream.Content = model.NewHTMLContent(txn.Content)

	// Save the new Stream to the database
	if err := streamService.Save(&stream, "Saved via Activity Intent"); err != nil {
		return derp.Wrap(err, location, "Error saving stream")
	}

	// Redirect to the "on-success" URL
	return ctx.Redirect(http.StatusSeeOther, onSuccess)
}

func GetIntent_Dislike(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil
}

func PostIntent_Dislike(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil

}

func GetIntent_Follow(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil

}

func PostIntent_Follow(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil

}

func GetIntent_Like(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil

}

func PostIntent_Like(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil
}

// WithUser handles boilerplate code for requests that require a signed-in user
func WithUser(serverFactory *server.Factory, fn func(ctx *steranko.Context, factory *domain.Factory, user *model.User) error) func(ctx echo.Context) error {

	const location = "handler.WithUser"

	return func(ctx echo.Context) error {

		// Cast the context to a Steranko Context
		sterankoContext, ok := ctx.(*steranko.Context)

		if !ok {
			return derp.NewInternalError(location, "Context must be a Steranko Context")
		}

		// Validate the domain name
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Guarantee that the user is signed in
		authorization := getAuthorization(ctx)

		if !authorization.IsAuthenticated() {
			return derp.NewUnauthorizedError(location, "You must be signed in to perform this action")
		}

		// Load the User from the database
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(authorization.UserID, &user); err != nil {
			return derp.Wrap(err, location, "Error loading user from database")
		}

		// Call the continuation function
		return fn(sterankoContext, factory, &user)
	}
}
