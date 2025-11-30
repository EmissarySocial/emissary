package service

import (
	"github.com/benpate/derp"
)

// Export service helps exports user data to another server
type Export struct {
	factory *Factory
}

// NewExport returns a fully populated Export service
func NewExport(factory *Factory) Export {
	return Export{
		factory: factory,
	}
}

func (service *Export) FindService(collectionName string) (Exportable, error) {

	const location = "service.Export.FindService"

	switch collectionName {
	case "outbox":
	case "content":
	case "following":
	case "blocked":

	case "emissary-annotation":
		return service.factory.Annotation(), nil

	case "emissary-circle":
		return service.factory.Circle(), nil

	case "emissary-conversation":
		return service.factory.Conversation(), nil

	case "emissary-folder":
		return service.factory.Folder(), nil

	case "emissary-follower":
		return service.factory.Follower(), nil

	case "emissary-following":
		return service.factory.Following(), nil

	case "emissary-inboxMessage":
		return service.factory.Inbox(), nil

	case "emissary-merchantAccount":
		return service.factory.MerchantAccount(), nil

	case "emissary-outboxMessage":
		return service.factory.Outbox(), nil

	case "emissary-privilege":
		return service.factory.Privilege(), nil

	case "emissary-product":
		return service.factory.Product(), nil

	case "emissary-response":
		return service.factory.Response(), nil

	case "emissary-rule":
		return service.factory.Rule(), nil

	case "emissary-stream":
		return service.factory.Stream(), nil

	case "emissary-user":
		return service.factory.User(), nil
	}

	return nil, derp.NotFound(location, "Collection not found", collectionName)
}
