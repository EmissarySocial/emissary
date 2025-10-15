package handler

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func GetGeocodeIP(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetGeocode"

	ipAddress := ctx.RealIP()
	geocodeService := factory.Geocode()

	latitude, longitude, err := geocodeService.GeocodeIP(session, ipAddress)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error retrieving geocode for IP address", ipAddress))
	}

	result := mapof.Any{
		"latitude":  latitude,
		"longitude": longitude,
	}

	return ctx.JSON(200, result)
}
