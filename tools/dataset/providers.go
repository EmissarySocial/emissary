package dataset

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/sliceof"
)

func Providers() sliceof.Object[form.LookupCode] {

	return []form.LookupCode{
		{
			Value:       "GIPHY",
			Label:       "Giphy",
			Icon:        "film",
			Description: "Free service (with attribution) for embedding GIF animations",
			Group:       "Images",
		},
		{
			Value:       "UNSPLASH",
			Label:       "Unsplash",
			Icon:        "picture",
			Description: "Free service (with attribution) embedding photos and images",
			Group:       "Images",
		},
		{
			Value:       model.ConnectionProviderArcGIS,
			Label:       "ArcGIS",
			Icon:        "globe",
			Description: "Commercial service with a free tier",
			Group:       "Geocode by Physical Address",
		},
		{
			Value:       model.ConnectionProviderGoogleMaps,
			Label:       "Google Maps",
			Icon:        "globe",
			Description: "Commercial service with a free tier",
			Group:       "Geocode by Physical Address",
		},
		{
			Value:       model.ConnectionProviderFREEIPAPICOM,
			Label:       "FREEIPAPI.COM",
			Icon:        "globe",
			Description: "Commercial service with a free tier",
			Group:       "Geocode by IP Address",
		},
		{
			Value:       model.ConnectionProviderIPAPICO,
			Label:       "IPAPI.CO",
			Icon:        "globe",
			Description: "Commercial service with a free tier",
			Group:       "Geocode by IP Address",
		},
		{
			Value:       model.ConnectionProviderIPAPICOM,
			Label:       "IP-API.COM",
			Icon:        "globe",
			Description: "Commercial service with a free tier",
			Group:       "Geocode by IP Address",
		},
		{
			Value:       model.ConnectionProviderStaticGeocoderIP,
			Label:       "Static Geocoder",
			Icon:        "globe",
			Description: "Return a fixed location for all IP addresses. Good for local communities.",
			Group:       "Geocode by IP Address",
		},
		{
			Value:       model.ConnectionProviderGeoapify,
			Label:       "Geoapify",
			Icon:        "globe",
			Description: "Commercia Geo-Search Service with Free Tier",
			Group:       "Geo-Search",
		},
		{
			Value:       model.ConnectionProviderNominatim,
			Label:       "Nominatim",
			Icon:        "globe",
			Description: "Open Source, Self-Hostable Geo-Search Service",
			Group:       "Geo-Search",
		},
		{
			Value:       model.ConnectionProviderStripe,
			Label:       "Stripe Payments",
			Icon:        "stripe",
			Description: "Users copy/paste API keys from their own Stripe Dashboard.",
			Group:       "User Payments",
		},
		{
			Value:       model.ConnectionProviderStripeConnect,
			Label:       "Stripe Connect",
			Icon:        "stripe",
			Description: "Users sign in via OAuth. Requires additional setup from admins.",
			Group:       "User Payments",
		},
	}
}
