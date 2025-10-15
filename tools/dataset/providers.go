package dataset

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
)

func Providers() []form.LookupCode {

	return []form.LookupCode{
		{
			Value:       "GIPHY",
			Label:       "Giphy",
			Icon:        "film",
			Description: "Embeddable GIF Images",
			Group:       "MANUAL",
		},
		{
			Value:       "UNSPLASH",
			Label:       "Unsplash",
			Icon:        "picture",
			Description: "Embeddable Photographs",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderArcGIS,
			Label:       "ArcGIS",
			Icon:        "globe",
			Description: "Geocoding for physical addresses and locations",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderGoogleMaps,
			Label:       "Google Maps",
			Icon:        "globe",
			Description: "Geocoding for physical addresses and locations",
			Group:       "MANUAL",
		},
		/*{
			Value:       model.ConnectionProviderGoogleMapsIP,
			Label:       "Google Maps (IP Geocoding)",
			Icon:        "globe",
			Description: "Geocoding for IP Addresses",
			Group:       "MANUAL",
		},*/
		{
			Value:       model.ConnectionProviderFREEIPAPICOM,
			Label:       "FREEIPAPI.COM",
			Icon:        "globe",
			Description: "Geocoding for IP Addresses",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderIPAPICO,
			Label:       "IPAPI.CO",
			Icon:        "globe",
			Description: "Geocoding for IP Addresses",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderIPAPICOM,
			Label:       "IP-API.COM",
			Icon:        "globe",
			Description: "Geocoding for IP Addresses",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderStaticGeocoderIP,
			Label:       "Static IP Geocoding",
			Icon:        "globe",
			Description: "Return a fixed location for all IP address geocoding requests.",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderStripe,
			Label:       "Stripe Payments",
			Icon:        "stripe",
			Description: "Users copy/paste API keys from their own Stripe Dashboard.",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderStripeConnect,
			Label:       "Stripe Connect",
			Icon:        "stripe",
			Description: "Users sign in via OAuth. Requires additional setup from admins.",
			Group:       "MANUAL",
		},
	}
}
