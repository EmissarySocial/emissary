// Package activitypub_domain serves the ActivityPub endpoints for the domain-wide "@search" actor:
// its JSON-LD profile and the outbox collection of everything the domain has indexed publicly.
//
// This is one actor for the whole domain, distinct from the per-query actors in
// activitypub_search and the per-user actors in activitypub_user.
package activitypub_domain

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetJSONLD generates JSON-LD for the @search domain actor
func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.GetJSONLD"

	// Generate JSON-LD for this @search domain actor
	searchDomainService := factory.SearchDomain()
	result, err := searchDomainService.GetJSONLD(session)

	if err != nil {
		return derp.Wrap(err, location, "Generating JSON-LD for search domain actor")
	}

	// Return success
	headers.SetVariant(ctx.Response().Header(), headers.VariantActivityPub)
	return ctx.JSON(200, result)
}
