package consumer

import (
	"github.com/benpate/turbine/queue"
)

// Consumer is the primary queue consumer for Emissary.  It handles background tasks that are triggered by the queue.
type Consumer struct {
	serverFactory ServerFactory
}

// New returns a fully initialized Consumer object
func New(serverFactory ServerFactory) Consumer {
	return Consumer{
		serverFactory: serverFactory,
	}
}

// Run is the actual consumer function that is called by the queue.
// It receives a task name and a map of arguments, and returns a boolean success value and an error.
func (consumer Consumer) Run(name string, args map[string]any) queue.Result {

	switch name {

	case "AddSearchResult":
		return WithFactory(consumer.serverFactory, args, AddSearchResult)

	case "CreateWebSubFollower":
		return WithFactory(consumer.serverFactory, args, CreateWebSubFollower)

	case "Geocode":
		return WithStream(consumer.serverFactory, args, Geocode)

	case "IndexAllStreams":
		return WithFactory(consumer.serverFactory, args, IndexAllStreams)

	case "IndexAllUsers":
		return WithFactory(consumer.serverFactory, args, IndexAllUsers)

	case "MakeStreamArchive":
		return WithStream(consumer.serverFactory, args, MakeStreamArchive)

	case "ProcessMedia":
		return WithFactory(consumer.serverFactory, args, ProcessMedia)

	case "ReceiveWebMention":
		return WithFactory(consumer.serverFactory, args, ReceiveWebMention)

	case "SendActivityPubMessage":
		return WithFactory(consumer.serverFactory, args, SendActivityPubMessage)

	case "SendWebMention":
		return SendWebMention(args)

	case "SendWebSubMessage":
		return SendWebSubMessage(args)

	case "stream.syndicate", "stream.syndicate.undo":
		return StreamSyndicate(name, args)

	}

	return queue.Ignored()
}
