package dataset

import "github.com/benpate/form"

func GeocodeTiles() []form.LookupCode {

	// https://wiki.openstreetmap.org/wiki/Raster_tile_providers

	return []form.LookupCode{
		{
			Group:       "Open Street Map",
			Label:       "Standard (English)",
			Value:       "OPEN-STREET-MAP-STANDARD",
			Description: "<a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
		},
		{
			Group:       "Open Street Map",
			Label:       "Standard (French)",
			Value:       "OPEN-STREET-MAP-FRENCH",
			Description: "<a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://a.tile.openstreetmap.fr/osmfr/{z}/{x}/{y}.png",
		},
		{
			Group:       "Open Street Map",
			Label:       "Standard (German)",
			Value:       "OPEN-STREET-MAP-GERMAN",
			Description: "<a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://tile.openstreetmap.de/{z}/{x}/{y}.png",
		},

		// https://apidocs.geoapify.com/docs/maps/geocode-tiles/
		{
			Group:       "Geoapify",
			Label:       "Bright",
			Value:       "GEOAPIFY-BRIGHT",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/osm-bright/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Bright (Grey)",
			Value:       "GEOAPIFY-BRIGHT-GREY",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/osm-bright-grey/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Bright (Smooth)",
			Value:       "GEOAPIFY-BRIGHT-SMOOTH",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/osm-bright-smooth/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Carto",
			Value:       "GEOAPIFY-CARTO",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/carto/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Liberty",
			Value:       "GEOAPIFY-LIBERTY",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/osm-liberty/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Maptiler 3D",
			Value:       "GEOAPIFY-MAPTILER-3D",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/maptiler-3d/{z}/{x}/{y}.png?apiKey={apiKey}",
		},
		{
			Group:       "Geoapify",
			Label:       "Positron",
			Value:       "GEOAPIFY-POSITRON",
			Description: "Powered by <a href=https://www.geoapify.com>Geoapify</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://maps.geoapify.com/v1/tile/positron/{z}/{x}/{y}.png?apiKey={apiKey}",
		},

		// https://www.maptiler.com
		{
			Group:       "Maptiler",
			Label:       "Basic",
			Value:       "MAPTILER-BASIC",
			Description: "<a href=https://maptiler.com/copyright>© MapTiler</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://api.maptiler.com/maps/basic/{z}/{x}/{y}.png?key={apiKey}",
		},
		{
			Group:       "Maptiler",
			Label:       "Outdoor",
			Value:       "MAPTILER-OUTDOOR",
			Description: "<a href=https://maptiler.com/copyright>© MapTiler</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://api.maptiler.com/maps/outdoor/{z}/{x}/{y}.png?key={apiKey}",
		},
		{
			Group:       "Maptiler",
			Label:       "Pastel",
			Value:       "MAPTILER-PASTEL",
			Description: "<a href=https://maptiler.com/copyright>© MapTiler</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://api.maptiler.com/maps/pastel/{z}/{x}/{y}.png?key={apiKey}",
		},
		{
			Group:       "Maptiler",
			Label:       "Streets",
			Value:       "MAPTILER-STREETS",
			Description: "Maps <a href=https://maptiler.com/copyright>© MapTiler</a> &middot Data <a href=https://openstreetmap.org/copyright>© OpenStreetMap contributors</a>",
			Href:        "https://api.maptiler.com/maps/streets/{z}/{x}/{y}.png?key={apiKey}",
		},

		// https://www.thunderforest.com/docs/geocode-tiles-api/
		{
			Group:       "Thunderforest",
			Label:       "Atlas",
			Value:       "THUNDERFOREST-ATLAS",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/atlas/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Landscape",
			Value:       "THUNDERFOREST-LANDSCAPE",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/landscape/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Neighbourhood",
			Value:       "THUNDERFOREST-NEIGHBOURHOOD",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/neighbourhood/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Open Cycle Map",
			Value:       "THUNDERFOREST-OPEN-CYCLE-MAP",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Outdoors",
			Value:       "THUNDERFOREST-OUTDOORS",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/outdoors/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Pioneer",
			Value:       "THUNDERFOREST-PIONEER",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/pioneer/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Transport",
			Value:       "THUNDERFOREST-TRANSPORT",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/transport/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group:       "Thunderforest",
			Label:       "Transport (Dark)",
			Value:       "THUNDERFOREST-TRANSPORT-DARK",
			Description: "Maps <a href=https://www.thunderforest.com>© Thunderforest</a> &middot; Data © <a href=https://www.openstreetmap.org/copyright>OpenStreetMap contributors</a>",
			Href:        "https://tile.thunderforest.com/transport-dark/{z}/{x}/{y}.png?apikey={apiKey}",
		},
		{
			Group: "Custom",
			Label: "Custom",
			Value: "CUSTOM",
			Href:  "{href}",
		},
	}
}
