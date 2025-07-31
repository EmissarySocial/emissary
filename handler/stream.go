package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub_search"
	"github.com/EmissarySocial/emissary/handler/activitypub_stream"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetStream handles GET requests with the default action.
// This handler also responds to JSON-LD requests
func GetStream(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	// Special case for JSON-LD requests.
	if isJSONLDRequest(ctx) {
		return getStreamJSONLD(ctx, factory, session, template, stream)
	}

	// Otherwise, just build the stream normally
	return getStreamPipeline(ctx, factory, session, template, stream, build.ActionMethodGet)
}

// GetStreamWithAction handles GET requests with a specified action
func GetStreamWithAction(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream) error {
	return getStreamPipeline(ctx, factory, session, template, stream, build.ActionMethodGet)
}

// PostStreamWithAction handles POST requests with a specified action
func PostStreamWithAction(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream) error {
	return getStreamPipeline(ctx, factory, session, template, stream, build.ActionMethodPost)
}

func getStreamJSONLD(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.getStreamJSONLD"

	// Special case for "search" templates
	if template.IsSearch() {

		// Locate/Create the SearchQuery
		searchQueryService := factory.SearchQuery()
		searchQuery, err := searchQueryService.LoadOrCreate(session, ctx.QueryParams())

		if err != nil {
			return derp.Wrap(err, location, "Error loading SearchQuery")
		}

		// Return JSON-LD for this search query
		return activitypub_search.GetJSONLD(ctx, factory, session, template, stream, &searchQuery)
	}

	// All other templates are "stream" templates
	return activitypub_stream.GetJSONLD(ctx, factory, session, template, stream)
}

// getStreamPipeline is the common Stream handler for both GET and POST requests
func getStreamPipeline(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, stream *model.Stream, actionMethod build.ActionMethod) error {

	const location = "handler.getStreamPipeline"

	// Try to find the action requested by the user.
	actionID := getActionID(ctx)

	// Get a stream builder.  This also enforces permissions
	streamBuilder, err := build.NewStream(factory, session, ctx.Request(), ctx.Response(), *template, stream, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to create Builder.")
	}

	// Add webmention link header per:
	// https://www.w3.org/TR/webmention/#sender-discovers-receiver-webmention-endpoint
	if actionMethod == build.ActionMethodGet {
		ctx.Response().Header().Set("Link", "/.webmention; rel=\"webmention\"")
	}

	// Build the HTML page (execute the pipeline)
	if err := build.AsHTML(ctx, factory, streamBuilder, actionMethod); err != nil {
		return derp.Wrap(err, location, "Unable to build page", stream.Token)
	}

	// Yusss
	return nil
}

// getStreamToken returns the :stream token from the Request (or a default)
func getStreamToken(ctx echo.Context) string {
	token := ctx.Param("stream")

	switch token {

	// Empty, or "zero" tokens just go to the home page instead
	case "", "000000000000000000000000":
		return "home"
	}

	return token
}
