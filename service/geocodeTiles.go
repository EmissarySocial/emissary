package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

type GeocodeTiles struct {
	connectionService *Connection
	hostname          string
}

func NewGeocodeTiles(connectionService *Connection, hostname string) GeocodeTiles {
	return GeocodeTiles{
		connectionService: connectionService,
		hostname:          hostname,
	}
}

func (service GeocodeTiles) GetTileURL(session data.Session) form.LookupCode {

	const location = "service.geocodeTiles.GetTileURL"

	// Try to load the map tile connection
	connection := model.NewConnection()

	if err := service.connectionService.LoadActiveByType(session, model.ConnectionTypeGeocodeTiles, &connection); err != nil {
		if !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Unable to load Map Tiles connection"))
		}
	}

	if style := connection.Data.GetString("style"); style != "" {
		lookupCodes := dataset.GeocodeTiles()

		for _, lookupCode := range lookupCodes {
			if lookupCode.Value == style {

				if lookupCode.Href == "{href}" {
					lookupCode.Href = connection.Data.GetString("href")
					return lookupCode
				}

				if strings.HasSuffix(lookupCode.Href, "{apiKey}") {
					apiKey := connection.Data.GetString("apiKey")
					lookupCode.Href = strings.ReplaceAll(lookupCode.Href, "{apiKey}", apiKey)
				}

				return lookupCode
			}
		}
	}

	return form.LookupCode{
		Value:       "OPEN-STREET-MAPS-STANDARD",
		Label:       "Standard (English)",
		Description: "<a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
		Href:        "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
	}
}
