package mailchimp

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
)

// Client calls the Mailchimp Marketing API on behalf of a single account
type Client struct {
	apiKey  string
	baseURL string
	options []remote.Option
}

// New returns a Client that addresses the provided data center using the provided credential
func New(apiKey string, dataCenter string, remoteOptions ...remote.Option) (Client, error) {

	const location = "tools.mailchimp.New"

	// RULE: a credential must survive an Authorization header before it is put into one
	if err := ValidateAPIKey(apiKey); err != nil {
		return Client{}, derp.Wrap(err, location, "Cannot use this Mailchimp API key")
	}

	// RULE: BaseURL validates the data center, which becomes a hostname
	baseURL, err := BaseURL(dataCenter)

	if err != nil {
		return Client{}, derp.Wrap(err, location, "Cannot address this Mailchimp data center")
	}

	return Client{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: baseURL,
		options: remoteOptions,
	}, nil
}

// get returns a Transaction that reads the named path from this Client's account
func (client Client) get(path string) *remote.Transaction {

	// Mailchimp accepts an API key or an OAuth token in this same header and cannot tell
	// them apart, which is the whole reason nothing here inspects the value.
	return remote.Get(client.baseURL + path).
		With(options.BearerAuth(client.apiKey)).
		With(options.Accept("application/json")).
		With(client.options...)
}
