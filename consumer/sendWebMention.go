package consumer

import (
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"willnorris.com/go/webmention"
)

func SendWebMention(args mapof.Any) queue.Result {

	const location = "consumer.SendWebMention"

	source := args.GetString("source") // nolint:scopeguard
	target := args.GetString("target") // nolint:scopeguard

	// Create a new HTTP client to send the webmentions
	client := webmention.New(nil)

	// RULE: No need to send web mentions to local domains
	if dt.IsLocalhost(target) {
		return queue.Success()
	}

	// Try to find endpont
	if endpoint, err := client.DiscoverEndpoint(target); err == nil {

		// RULE: Do not allow remote servers to send webmentions to local domain either
		if dt.IsLocalhost(endpoint) {
			return queue.Success()
		}

		if response, err := client.SendWebmention(endpoint, source, target); err != nil {
			return queue.Error(derp.Wrap(err, location, "Error sending webmention", source, target, response))
		}
	}

	// Veni vidi vici
	return queue.Success()
}
