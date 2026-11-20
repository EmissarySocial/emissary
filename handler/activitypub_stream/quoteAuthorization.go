package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetQuoteAuthorization serves the FEP-044f QuoteAuthorization that approves one quote of a Stream.
func GetQuoteAuthorization(ctx *steranko.Context, factory *service.Factory, session data.Session, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetQuoteAuthorization"

	// Mastodon re-fetches every QuoteAuthorization weekly, and a revoked one is an expected 404,
	// so the 404s below are answered directly rather than reported to the error log.

	// Parse the QuoteAuthorization ID from the URL
	quoteAuthorizationID, err := primitive.ObjectIDFromHex(ctx.Param("quoteAuthorizationId"))

	if err != nil {
		return ctx.NoContent(http.StatusNotFound)
	}

	// Load the approved quote.  A Stream that no longer accepts quotes serves none of them
	quote := model.NewCollectionItem()

	if err := factory.Stream().LoadQuote(session, stream, quoteAuthorizationID, &quote); err != nil {

		if derp.IsNotFound(err) {
			return ctx.NoContent(http.StatusNotFound)
		}

		return derp.Wrap(err, location, "Loading quote", stream.StreamID, quoteAuthorizationID)
	}

	// Serve the QuoteAuthorization
	ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)
	return ctx.JSON(http.StatusOK, stream.QuoteAuthorizationJSONLD(quote))
}
