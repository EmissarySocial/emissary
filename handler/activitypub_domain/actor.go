package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

// GetJSONLD generates JSON-LD for the @search domain actor
func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.GetJSONLD"

	// Generate JSON-LD for this @search domain actor
	searchDomainService := factory.SearchDomain()
	result, err := searchDomainService.GetJSONLD(session)

	if err != nil {
		return derp.Wrap(err, location, "Unable to generate JSON-LD for search domain actor")
	}

	// Return success
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}
