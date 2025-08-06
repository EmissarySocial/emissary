package consumer

import (
	"context"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// WithFactory wraps a consumer function, and uses the "host" argument to inject a Factory object into the function signature.
func WithFactory(serverFactory ServerFactory, args mapof.Any, handler func(factory *domain.Factory, session data.Session, args mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithFactory"

	// Get the host argument from the map
	hostname := args.GetString("host")
	hostname = dt.NameOnly(hostname)

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

	// Execute the handler as a transaction
	result, err := factory.Server().WithTransaction(context.TODO(), func(session data.Session) (any, error) {
		result := handler(factory, session, args)
		return result, result.Error
	})

	// Return the queue result
	if result, isQueueResult := result.(queue.Result); isQueueResult {
		return result
	}

	// Guard against panics if developers do bad things.  This should never happen.
	return queue.Failure(derp.InternalError(location, "Handler did not return a queue.Result.  This should never happen", result))
}

// WithStream wraps a consumer function, using the "streamId" argument to load a Stream object from the database.
func WithStream(serverFactory ServerFactory, args mapof.Any, handler func(*domain.Factory, data.Session, *service.Stream, *model.Stream, mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithStream"

	return WithFactory(serverFactory, args, func(factory *domain.Factory, session data.Session, args mapof.Any) queue.Result {

		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByToken(session, args.GetString("streamId"), &stream); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load stream", args))
		}

		return handler(factory, session, streamService, &stream, args)
	})
}
