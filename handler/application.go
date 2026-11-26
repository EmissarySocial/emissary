package handler

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

// GetApplicationActor generates JSON-LD for the @application actor
func GetApplicationActor(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetApplicationActor"

	// Generate JSON-LD for this @application actor
	domainService := factory.Domain()
	result, err := domainService.GetJSONLD(session)

	if err != nil {
		return derp.Wrap(err, location, "Generating JSON-LD for domain actor")
	}

	// Return Success
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}
