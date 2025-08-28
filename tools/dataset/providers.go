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
			Description: "Look up addresses and locations",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderGoogleMaps,
			Label:       "Google Maps",
			Icon:        "globe",
			Description: "Look up addresses and locations",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderStripe,
			Label:       "Stripe Payments",
			Icon:        "stripe",
			Description: "Allow users to accept payments via Stripe. Users copy/paste API keys directly from their own Stripe Dashboard. (Easier for Server Admins)",
			Group:       "MANUAL",
		},
		{
			Value:       model.ConnectionProviderStripeConnect,
			Label:       "Stripe Connect",
			Icon:        "stripe",
			Description: "Allow users to accept payments via Stripe.  Use this configuration if you have set up Stripe Connect and want users to authenticate via OAuth. (Easier for Users)",
			Group:       "MANUAL",
		},
	}
}
