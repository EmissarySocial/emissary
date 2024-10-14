package consumer

import (
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"willnorris.com/go/webmention"
)

func SendWebMention(args mapof.Any) error {

	const location = "consumer.SendWebMention"

	source := args.GetString("source")
	target := args.GetString("target")

	// Create a new HTTP client to send the webmentions
	client := webmention.New(nil)

	// RULE: No need to send web mentions to local domains
	if domain.IsLocalhost(target) {
		return nil
	}

	// Try to find endpont
	if endpoint, err := client.DiscoverEndpoint(target); err == nil {

		// RULE: Do not allow remote servers to send webmentions to local domain either
		if domain.IsLocalhost(endpoint) {
			return nil
		}

		if response, err := client.SendWebmention(endpoint, source, target); err != nil {
			return derp.Wrap(err, location, "Error sending webmention", source, target, response)
		}
	}

	// Veni vidi vici
	return nil
}
