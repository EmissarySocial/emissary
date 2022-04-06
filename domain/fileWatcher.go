package domain

import (
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/service"
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
