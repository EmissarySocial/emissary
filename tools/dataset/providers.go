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
			Group:       "OAUTH",
		},

		/* REMOVED FOR NOW
		{
			Value:       model.ConnectionProviderPayPal,
			Label:       "PayPal Marketplace",
			Icon:        "paypal",
			Description: "(On Probation) Allows users to accept payments via PayPal. Requires a Marketplace account with PayPal.",
			Group:       "MANUAL",
		},
		{
			Value:       "SQUARE",
			Label:       "Square",
			Icon:        "square",
			Description: "Allow users to accept payments via Square.",
			Group:       "MANUAL",
			Href:        "https://developer.squareup.com",
		},
		{
			Value:       "SHOPIFY",
			Label:       "Shopify",
			Icon:        "shopify",
			Description: "Allow users to accept payments via Shopify.",
			Group:       "MANUAL",
			Href:       "https://shopify.dev",
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
