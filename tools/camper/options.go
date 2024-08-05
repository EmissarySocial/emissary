package camper

import (
	"net/http"

	"github.com/benpate/remote/options"
)

type Option func(*Camper)

func WithClient(client *http.Client) Option {
	return func(camper *Camper) {
		camper.options = append(camper.options, options.WithClient(client))
	}
}
