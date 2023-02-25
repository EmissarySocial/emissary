package tasks

import (
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"willnorris.com/go/webmention"
)

type SendWebMention struct {
	source string // URL of the internal document that is linking
	target string // URL of the external document being linked to
}

func NewSendWebMention(source string, target string) SendWebMention {

	return SendWebMention{
		source: source,
		target: target,
	}
}

func (task SendWebMention) Run() error {

	// Create a new HTTP client to send the webmentions
	client := webmention.New(nil)

	// RULE: No need to send web mentions to local domains
	if domain.IsLocalhost(task.target) {
		return nil
	}

	// Try to find endpont
	if endpoint, err := client.DiscoverEndpoint(task.target); err == nil {

		// RULE: Do not allow remote servers to send webmentions to local domain either
		if domain.IsLocalhost(endpoint) {
			return nil
		}

		if response, err := client.SendWebmention(endpoint, task.source, task.target); err != nil {
			return derp.Wrap(err, "mention.SendWebMention.Run", "Error sending webmention", task, response)
		}
	}

	// Veni vidi vici
	return nil
}
