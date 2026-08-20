package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

// GetStartup renders a page of the first-run setup checklist
func GetStartup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return doStartup(ctx, factory, session, build.ActionMethodGet)
}

// PostStartup applies an action from the first-run setup checklist
func PostStartup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return doStartup(ctx, factory, session, build.ActionMethodPost)
}

// doStartup renders or applies a setup checklist action, for a domain owner only
func doStartup(ctx *steranko.Context, factory *service.Factory, session data.Session, method build.ActionMethod) error {

	const location = "handler.doStartup"

	// Only domain owners can access admin pages
	if !isOwner(ctx.Authorization()) {
		return derp.Unauthorized(location, "Unauthorized")
	}

	// RULE: The startup checklist is only available while the Domain is still being set up.  Once the
	if domain := factory.Domain().Get(); domain.StateID != model.DomainStateStartup {
		return ctx.Redirect(http.StatusPermanentRedirect, "/")
	}

	// Collect parameters to build
	templateService := factory.Template()
	template, err := templateService.LoadAdmin("startup")

	if err != nil {
		return derp.Wrap(err, location, "Loading template")
	}

	actionID := first.String(ctx.Param("action"), "page")

	// Get a Builder for this page (also authenticates admin permissions)
	builder, err := build.NewDomain(factory, session, ctx.Request(), ctx.Response(), template, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Creating builder")
	}

	// Build the HTML
	if err := build.AsHTML(ctx, factory, builder, method); err != nil {
		return derp.Wrap(err, location, "Building page")
	}

	return nil
}
