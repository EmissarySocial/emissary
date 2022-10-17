package external

import "github.com/stripe/stripe-go/v72/client"

type Factory interface {
	StripeClient() (client.API, error)
	Hostname() string
}
