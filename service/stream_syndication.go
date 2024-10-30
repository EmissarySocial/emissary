package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// sendSyndicationMessages sends messages to syndication targets
// whenever a stream is added or removed from syndication
func (service *Stream) sendSyndicationMessages(stream *model.Stream, added []string, removed []string) error {

	const location = "service.Stream.sendSyndicationMessages"

	domain := service.domainService.Get()
	message := stream.GetWebhookData()

	for _, target := range domain.Syndication {

		// Send syndication messages
		for _, endpoint := range added {
			if target.Value == endpoint {

				message.SetString("event", "stream.syndicate")
				task := queue.NewTask("stream.syndicate.undo", mapof.Any{
					"endpoint": target.Href,
					"message":  message,
				})

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing syndication undo task", task)
				}
				break
			}
		}

		// Send syndication:undo messages
		for _, endpoint := range removed {
			if target.Value == endpoint {

				message.SetString("event", "stream.syndicate:undo")
				task := queue.NewTask("stream.syndicate.undo", mapof.Any{
					"endpoint": target.Href,
					"message":  message,
				})

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing syndication undo task", task)
				}
				break
			}
		}
	}

	return nil
}
