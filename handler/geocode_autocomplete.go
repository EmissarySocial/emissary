package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/geo"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
)

func GetGeocodeAutocomplete(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetGeocodeAutocomplete"

	// Query the Autocomplete service (allow any parameter to work..)
	query := ctx.QueryParam("q")

	// If we have an empty query, then return empty results
	if query == "" {
		return ctx.NoContent(http.StatusOK)
	}

	bias := geo.Point{}

	// Calculate Bias Point (if present)
	if longitudeString := ctx.QueryParam("longitude"); longitudeString != "" {

		if latitudeString := ctx.QueryParam("latitude"); latitudeString != "" {

			longitude := convert.Float(longitudeString)
			latitude := convert.Float(latitudeString)

			if (longitude != 0) || (latitude != 0) {
				bias.Longitude = longitude
				bias.Latitude = latitude
			}
		}
	}

	// Use the Autocomplete service to search for places matching this query
	autocompleteService := factory.GeocodeAutocomplete()
	results, err := autocompleteService.Search(session, query, bias)

	if err != nil {
		return derp.Wrap(err, location, "Unable to search geo database", derp.WithCode(500))
	}

	// Output the results as <option value="xx" data-latitude="xx" data-longitude="xx">Xxx</option>
	b := html.New()

	b.Div().Class("menu", "border", "rounded-bottom")

	if results.IsEmpty() {
		b.Div().Class("padding").InnerText("No places found that match your search.")
	}

	for _, result := range results {
		b.Div().
			Role("menuitem").
			Data("latitude", convert.String(result.Latitude)).
			Data("longitude", convert.String(result.Longitude)).
			Script("on click trigger Select(place:this)").
			InnerText(result.Name).
			Close()
	}

	b.CloseAll()

	// Station.
	return ctx.HTML(http.StatusOK, b.String())
}
