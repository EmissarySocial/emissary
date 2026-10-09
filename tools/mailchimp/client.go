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
	return client.authorize(remote.Get(client.baseURL + path))
}

// post returns a Transaction that creates something in this Client's account
func (client Client) post(path string, body any) *remote.Transaction {
	return client.authorize(remote.Post(client.baseURL + path).JSON(body))
}

// put returns a Transaction that creates or replaces something in this Client's account
func (client Client) put(path string, body any) *remote.Transaction {
	return client.authorize(remote.Put(client.baseURL + path).JSON(body))
}

// patch returns a Transaction that partially updates something in this Client's account
func (client Client) patch(path string, body any) *remote.Transaction {
	return client.authorize(remote.Patch(client.baseURL + path).JSON(body))
}

// delete returns a Transaction that removes the named path from this Client's account
func (client Client) delete(path string) *remote.Transaction {
	return client.authorize(remote.Delete(client.baseURL + path))
}

// authorize applies this Client's credential and options to a Transaction
func (client Client) authorize(transaction *remote.Transaction) *remote.Transaction {

	return transaction.
		With(options.BearerAuth(client.apiKey)).
		With(options.Accept("application/json")).
		With(client.options...)
}
