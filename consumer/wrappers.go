package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	domainTools "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// WithFactory wraps a consumer function, and uses the "host" argument to inject a Factory object into the function signature.
func WithFactory(serverFactory ServerFactory, args mapof.Any, Handler func(factory *domain.Factory, args mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithFactory"

	// Get the host argument from the map
	hostname := args.GetString("host")
	hostname = domainTools.NameOnly(hostname)

	if hostname == "" {
		// If we don't have a host, we'll never be able to run this task, so hard fail
		return queue.Failure(derp.InternalError(location, "Missing 'host' argument"))
	}

	// Load the factory
	factory, err := serverFactory.ByHostname(hostname)

	if err != nil {
		// If we can't load the factory, maybe we can in the future, so try again.
		return queue.Failure(derp.Wrap(err, location, "Invalid 'host' argument.", hostname))
	}

	// Execute the handler with the Factory
	return Handler(factory, args)
}

// WithStream wraps a consumer function, using the "streamId" argument to load a Stream object from the database.
func WithStream(serverFactory ServerFactory, args mapof.Any, Handler func(*domain.Factory, *service.Stream, *model.Stream, mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithStream"

	return WithFactory(serverFactory, args, func(factory *domain.Factory, args mapof.Any) queue.Result {

		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByToken(args.GetString("streamId"), &stream); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load stream", args))
		}

		return Handler(factory, streamService, &stream, args)
	})
}
