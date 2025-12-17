package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

// sendSyndicationMessages sends messages to syndication targets
// whenever a stream is added or removed from syndication
func (service *Stream) sendSyndicationMessages(stream *model.Stream, added []string, changed []string, removed []string) error {

	domain := service.domainService.Get()
	object := stream.GetWebhookData()

	for _, target := range domain.Syndication {

		// Send syndication messages
		for _, endpoint := range added {

			if target.Value != endpoint {
				continue
			}

			service.queue.NewTask(
				"syndication.create",
				mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Create",
						"object": object,
					},
				},
			)
		}

		// Send syndication:undo messages
		for _, endpoint := range changed {

			if target.Value != endpoint {
				continue
			}

			service.queue.NewTask(
				"syndication.update",
				mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Update",
						"object": object,
					},
				},
			)
		}

		// Send syndication:undo messages
		for _, endpoint := range removed {

			if target.Value != endpoint {
				continue
			}

			service.queue.NewTask(
				"syndication.delete",
				mapof.Any{
					"endpoint": target.Href,
					"message": mapof.Any{
						"type":   "Delete",
						"object": object,
					},
				},
			)
		}
	}

	return nil
}
