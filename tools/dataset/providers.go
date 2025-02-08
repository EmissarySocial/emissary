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

		/* REMOVED FOR NOW
		{
			Value:       "STRIPE",
			Label:       "Stripe",
			Icon:        "stripe",
			Description: "To migrate from original API key",
			Group:       "MANUAL",
		},
		{
			Value:       "FACEBOOK",
			Label:       "Facebook",
			Icon:        "facebook",
			Description: "TBD",
			Group:       "OAUTH",
		},
		{
			Value:       "INSTAGRAM",
			Label:       "Instagram",
			Icon:        "instagram",
			Description: "TBD",
			Group:       "OAUTH",
		},
		{
			Value:       "LINKEDIN",
			Label:       "LinkedIn",
			Icon:        "linkedin",
			Description: "TBD",
			Group:       "OAUTH",
		},
		{
			Value:       "TWITTER",
			Label:       "Twitter",
			Icon:        "twitter",
			Description: "Link to Twitter data feeds",
			Group:       "OAUTH",
		},
		*/
	}
}
