package service

import (
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"willnorris.com/go/webmention"
)

type TaskSendWebMention struct {
	source string // URL of the internal document that is linking
	target string // URL of the external document being linked to
}

func NewTaskSendWebMention(source string, target string) TaskSendWebMention {

	return TaskSendWebMention{
		source: source,
		target: target,
	}
}

func (task TaskSendWebMention) Run() error {

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
			return derp.Wrap(err, "mention.TaskSendWebMention.Run", "Error sending webmention", task, response)
		}
	}

	// Veni vidi vici
	return nil
}
