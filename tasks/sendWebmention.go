package tasks

import (
	"strings"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"willnorris.com/go/webmention"
)

type SendWebMention struct {
	mentionService *service.Mention // WebMention service provides useful helper function.
	source         string           // URL of the source document
	html           string           // HTML contents of the source document (to be parsed for links)
}

func NewSendWebMention(mentionService *service.Mention, source string, html string) SendWebMention {

	return SendWebMention{
		mentionService: mentionService,
		source:         source,
		html:           html,
	}
}

func (task SendWebMention) Run() error {

	reader := strings.NewReader(task.html)
	links, err := webmention.DiscoverLinksFromReader(reader, task.source, "")

	if err != nil {
		return derp.Wrap(err, "mention.SendWebMention.Run", "Error discovering webmention links", task)
	}

	if len(links) == 0 {
		return nil
	}

	// Create a new HTTP client to send the webmentions
	client := webmention.New(nil)

	// Send webmentions
	for _, target := range links {

		// RULE: No need to send web mentions to local domains
		if domain.IsLocalhost(target) {
			continue
		}

		// Try to find endpont
		if endpoint, err := client.DiscoverEndpoint(target); err == nil {

			// RULE: Do not allow remote servers to send webmentions to local domain either
			if domain.IsLocalhost(endpoint) {
				continue
			}

			if response, err := client.SendWebmention(endpoint, task.source, target); err != nil {
				derp.Report(derp.Wrap(err, "mention.SendWebMention.Run", "Error sending webmention", task, response))
				continue
			}
		}

		// TODO: MEDIUM: how to handle errors?  Retry the task later?  How many times?
	}

	return nil
}
