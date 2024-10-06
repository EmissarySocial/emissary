package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"willnorris.com/go/webmention"
)

type TaskSendWebMention struct {
	Source string // URL of the internal document that is linking
	Target string // URL of the external document being linked to
}

func NewTaskSendWebMention(source string, target string) TaskSendWebMention {

	return TaskSendWebMention{
		Source: source,
		Target: target,
	}
}

func (task TaskSendWebMention) Priority() int {
	return 20
}

func (task TaskSendWebMention) RetryMax() int {
	return 12 // 4096 minutes = 68 hours ~= 3 days
}

func (task TaskSendWebMention) Hostname() string {
	return domain.NameOnly(task.Target)
}

func (task TaskSendWebMention) MarshalMap() map[string]any {
	return mapof.Any{
		"source": task.Source,
		"target": task.Target,
	}
}

func (task TaskSendWebMention) Run() error {

	// Create a new HTTP client to send the webmentions
	client := webmention.New(nil)

	// RULE: No need to send web mentions to local domains
	if domain.IsLocalhost(task.Target) {
		return nil
	}

	// Try to find endpont
	if endpoint, err := client.DiscoverEndpoint(task.Target); err == nil {

		// RULE: Do not allow remote servers to send webmentions to local domain either
		if domain.IsLocalhost(endpoint) {
			return nil
		}

		if response, err := client.SendWebmention(endpoint, task.Source, task.Target); err != nil {
			return derp.Wrap(err, "mention.TaskSendWebMention.Run", "Error sending webmention", task, response)
		}
	}

	// Veni vidi vici
	return nil
}
