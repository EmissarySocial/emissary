package dataset

import (
	"github.com/benpate/form"
)

func Providers() []form.LookupCode {

	return []form.LookupCode{
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
			Value:       "STRIPE",
			Label:       "Stripe",
			Icon:        "stripe",
			Description: "To migrate from original API key",
			Group:       "MANUAL",
		},
		{
			Value:       "TWITTER",
			Label:       "Twitter",
			Icon:        "twitter",
			Description: "Link to Twitter data feeds",
			Group:       "OAUTH",
		},
	}
}
