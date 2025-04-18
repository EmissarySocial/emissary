package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	builder "github.com/benpate/exp-builder"
	"github.com/labstack/echo/v4"
)

// TODO: Remove?
func ModelService_RenderTemplate(serverFactory *server.Factory, ctx echo.Context, serviceName string, builder builder.Builder, templateName string) (string, error) {

	const location = "handler.ModelService_RenderTemplate"

	// Try to find the factory for this hostname
	factory, err := serverFactory.ByContext(ctx)

	if err != nil {
		return "", derp.Wrap(err, location, "Invalid Hostname")
	}

	// Require that the user is signed in
	authorization := getAuthorization(ctx)

	if !authorization.IsAuthenticated() {
		return "", derp.NewUnauthorizedError(location, "You must be signed in to continue")
	}

	// Get the model service
	modelService, err := factory.Model(serviceName)

	if err != nil {
		return "", derp.Wrap(err, location, "Cannot make model service", serviceName)
	}

	// Use criteria builder to create a query
	criteria, err := builder.EvaluateAll(ctx.QueryParams())

	if err != nil {
		return "", derp.NewNotFoundError(location, "Cannot evaluate query parameters", err)
	}

	// Try to load the object from the database
	object, err := modelService.ObjectLoad(criteria)

	if err != nil {
		return "", derp.NewNotFoundError(location, "Cannot load object", err)
	}

	// Verify Permissions
	if err := modelService.ObjectUserCan(object, authorization, "view"); err != nil {
		return "", derp.Wrap(err, location, "You are not authorized to view this object")
	}

	// TODO: Render the template??
	return "", nil
}

func ModelService_Form(ctx echo.Context, modelService service.ModelService) {

}
