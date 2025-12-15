package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetApplicationActor generates JSON-LD for the @application actor
func GetApplicationActor(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetApplicationActor"

	// Generate JSON-LD for this @application actor
	domainService := factory.Domain()
	result, err := domainService.GetJSONLD(session)

	if err != nil {
		return derp.Wrap(err, location, "Unable to generate JSON-LD for domain actor")
	}

	// Return Success
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}

// PostApplicationActor_Inbox does not take any actions, but only logs the request
// IF logger is in Debug or Trace mode.
func PostApplicationActor_Inbox(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	}
}
