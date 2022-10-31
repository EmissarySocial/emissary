package tasks

import (
	"strings"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
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

	spew.Dump("tasks.SendWebMention.Run", task.source, task.html)
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

	spew.Dump("found links", links)

	// Send webmentions
	for _, target := range links {

		// TODO: Add filter for local domains...

		spew.Dump("processing link", target)

		// Try to find endpont
		if endpoint, err := client.DiscoverEndpoint(target); err == nil {

			spew.Dump("endpoint found", endpoint, "")

			response, err := client.SendWebmention(endpoint, task.source, target)

			if err != nil {
				spew.Dump(err)
				derp.Report(derp.Wrap(err, "mention.SendWebMention.Run", "Error sending webmention", task, response))
			}

			var buffer []byte
			response.Body.Read(buffer)
			spew.Dump(response.StatusCode, response.Status, string(buffer))

		} else {
			spew.Dump(err)
		}

		// TODO: how to handle errors?  Retry the task later?  How many times?
	}

	return nil
}
