package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostEmailFollower(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostEmailFollower"

	transaction := struct {
		ParentID primitive.ObjectID `form:"parentId"`
		Type     string             `form:"type"`
		Name     string             `form:"name"`
		Email    string             `form:"email"`
	}{}

	// Collect inputs from the context
	if err := ctx.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Unable to bind input")
	}

	// Generate follower secret (doing this first because it shouldn't fail,
	// but if it does, we want to fail here before we hit the database)
	secret, err := random.GenerateString(64)

	if err != nil {
		return derp.Wrap(err, location, "Error generating secret")
	}

	// Create the new "Follower" record.
	// Save the follower
	followerService := factory.Follower()
	follower, err := followerService.LoadOrCreate(session, transaction.ParentID, transaction.Email)

	if err != nil {
		return derp.Wrap(err, location, "Error saving follower")
	}

	follower.ParentType = transaction.Type
	follower.ParentID = transaction.ParentID
	follower.Method = model.FollowerMethodEmail
	follower.Format = model.MimeTypeHTML
	follower.Actor.ProfileURL = transaction.Email
	follower.Actor.EmailAddress = transaction.Email
	follower.Actor.Name = transaction.Name
	follower.Data.SetString("secret", secret)

	// Only reset the status if this is a new follower.  Otherwise,
	// this subscription may already be "ACTIVE" and we don't want to
	// roll back if we don't have to.
	if follower.IsNew() {
		follower.StateID = model.FollowerStatePending
	}

	if err := followerService.Save(session, &follower, "Email Follower signup"); err != nil {
		return derp.Wrap(err, location, "Error saving follower")
	}

	if err := followerService.SendFollowConfirmation(session, &follower); err != nil {
		return derp.Wrap(err, location, "Error sending confirmation email")
	}

	// Forward the user to the confirmation page
	return ctx.Redirect(http.StatusFound, "/@"+follower.ParentID.Hex()+"/follow-email-sent")
}
