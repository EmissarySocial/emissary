package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAPIActors returns a list of actors that match the provided search criteria.
// This is used in the E2EE service, as well as other actor lookups
func GetAPIActors(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetAPIActors"

	activityService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)

	searchString := ctx.QueryParam("q")
	actors, err := activityService.QueryActors(searchString)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query actors", searchString)
	}

	return ctx.JSON(http.StatusOK, actors)
}
