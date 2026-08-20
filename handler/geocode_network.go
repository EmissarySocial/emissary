package handler

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetGeocodeNetwork returns the approximate coordinates of the caller, based on their IP address
func GetGeocodeNetwork(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetGeocode"

	ipAddress := factory.ClientIP(ctx.Request())
	geocodeService := factory.GeocodeNetwork()

	point, err := geocodeService.Geocode(session, ipAddress)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Retrieving geocode for IP address", ipAddress))
	}

	result := mapof.Any{
		"longitude": point.Longitude,
		"latitude":  point.Latitude,
	}

	return ctx.JSON(200, result)
}
