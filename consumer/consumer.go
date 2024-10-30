package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
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
func (consumer Consumer) Run(name string, args map[string]any) (bool, error) {

	var factory *domain.Factory

	arguments := mapof.Any(args)

	if host := arguments.GetString("host"); host != "" {
		var err error
		factory, err = consumer.serverFactory.ByDomainName(host)

		if err != nil {
			return true, derp.Wrap(err, "consumer.Run", "Cannot find factory for host", host)
		}
	}

	switch name {

	case "CreateWebSubFollower":
		return true, CreateWebSubFollower(factory, args)

	case "ReceiveWebMention":
		return true, ReceiveWebMention(factory, args)

	case "SendActivityPubMessage":
		return true, SendActivityPubMessage(factory, args)

	case "SendWebMention":
		return true, SendWebMention(args)

	case "SendWebSubMessage":
		return true, SendWebSubMessage(args)

	case "stream.syndicate", "stream.syndicate.undo":
		return true, StreamSyndicate(args)

	}

	return false, nil
}
