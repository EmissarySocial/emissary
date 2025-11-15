package dataset

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/sliceof"
)

func Providers() sliceof.Object[form.LookupCode] {

	return []form.LookupCode{
		{
			Value:       model.ConnectionProviderGiphy,
			Label:       "Giphy",
			Icon:        "film",
			Description: "Free service (with attribution) for embedding GIF animations",
			Group:       "Images",
		},
		{
			Value:       model.ConnectionProviderUnsplash,
			Label:       "Unsplash",
			Icon:        "picture",
			Description: "Free service (with attribution) embedding photos and images",
			Group:       "Images",
		},
		{
			Value:       model.ConnectionProviderGeocodeAutocomplete,
			Label:       "Address Search",
			Icon:        "autocomplete",
			Description: "Search for addresses and place names in a global address book",
			Group:       "Mapping Services",
		},
		{
			Value:       model.ConnectionProviderGeocodeAddress,
			Label:       "Address Geocoder",
			Icon:        "geolocate",
			Description: "Find Lat/Lng coordinates from a street address",
			Group:       "Mapping Services",
		},
		{
			Value:       model.ConnectionProviderGeocodeNetwork,
			Label:       "Network Geocoder",
			Icon:        "network",
			Description: "Find Lat/Lng coordinates from a network (IP) address",
			Group:       "Mapping Services",
		},
		{
			Value:       model.ConnectionProviderGeocodeTiles,
			Label:       "Map Tiles",
			Icon:        "map",
			Description: "Display custom map layers",
			Group:       "Mapping Services",
		},
		{
			Value:       model.ConnectionProviderGeocodeTimezone,
			Label:       "Timezone Geocoder",
			Icon:        "clock",
			Description: "Look up local timezones for specific addresses",
			Group:       "Mapping Services",
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
