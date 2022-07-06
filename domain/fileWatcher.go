package domain

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
)

// WatchTemplates may get removed.  I dunno.
func WatchTemplates(streamService *service.Stream, streamUpdates chan model.Stream, templateUpdates chan model.Template) {

	for {

		template := <-templateUpdates

		streams, err := streamService.ListByTemplate(template.TemplateID)

		if err != nil {
			derp.Report(derp.Wrap(err, "domain.WatchTemplates", "Error retrieving streams that match templateID", template.TemplateID))
		}

		var stream model.Stream
		for streams.Next(&stream) {
			streamUpdates <- stream
		}
	}
}
