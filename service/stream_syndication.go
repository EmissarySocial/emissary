package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// sendSyndicationMessages sends messages to syndication targets
// whenever a stream is added or removed from syndication
func (service *Stream) sendSyndicationMessages(stream *model.Stream, added []string, changed []string, removed []string) error {

	const location = "service.Stream.sendSyndicationMessages"

	domain := service.domainService.Get()
	object := stream.GetWebhookData()

	for _, target := range domain.Syndication {

		// Send syndication messages
		for _, endpoint := range added {
			if target.Value == endpoint {

				task := queue.NewTask("syndication.create", mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Create",
						"object": object,
					},
				})

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing syndication undo task", task)
				}
			}
		}

		// Send syndication:undo messages
		for _, endpoint := range changed {
			if target.Value == endpoint {
				task := queue.NewTask("syndication.update", mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Update",
						"object": object,
					},
				})

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing syndication undo task", task)
				}
			}
		}

		// Send syndication:undo messages
		for _, endpoint := range removed {
			if target.Value == endpoint {
				task := queue.NewTask("syndication.delete", mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Delete",
						"object": object,
					},
				})

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing syndication undo task", task)
				}
			}
		}
	}

	return nil
}
