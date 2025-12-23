package consumer

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WithFactory wraps a consumer function, and uses the "host" argument to inject a Factory object into the function signature.
func WithFactory(serverFactory ServerFactory, args mapof.Any, handler func(factory *service.Factory, args mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithFactory"

	// Get the host argument from the map
	hostname := args.GetString("host")
	hostname = dt.NameOnly(hostname)

	if hostname == "" {
		// If we don't have a host, we'll never be able to run this task, so hard fail
		return queue.Failure(derp.InternalError(location, "Missing 'host' argument", args))
	}

	// Load the factory
	factory, err := serverFactory.ByHostname(hostname)

	if err != nil {
		// If we can't load the factory, maybe we can in the future, so try again.
		return queue.Error(derp.Wrap(err, location, "Invalid 'host' argument.", args))
	}

	return handler(factory, args)
}

// WithImport wraps a consumer function, using the "streamId" argument to load a Import object from the database.
func WithImport(serverFactory ServerFactory, args mapof.Any, handler func(*service.Factory, data.Session, *model.User, *model.Import, mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithImport"

	return WithUser(serverFactory, args, func(factory *service.Factory, session data.Session, user *model.User, args mapof.Any) queue.Result {

		importService := factory.Import()
		record := model.NewImport()
		importToken := args.GetString("importId")

		// Parse the UserID
		userToken := args.GetString("userId")
		userID, err := primitive.ObjectIDFromHex(userToken)

		if err != nil {
			return queue.Failure(derp.Wrap(err, location, "Unable to parse userID", userToken))
		}

		// Load the Import record
		if err := importService.LoadByToken(session, userID, importToken, &record); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load import", args))
		}

		return handler(factory, session, user, &record, args)
	})
}

// WithSession wraps a queue consumer function, and creates new database transaction that is passed to the wrapped handler
func WithSession(serverFactory ServerFactory, args mapof.Any, handler func(factory *service.Factory, session data.Session, args mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithFactoryAndSession"

	return WithFactory(serverFactory, args, func(factory *service.Factory, args mapof.Any) queue.Result {

		// Execute the handler as a transaction
		result, err := factory.Server().WithTransaction(context.Background(), func(session data.Session) (any, error) {
			result := handler(factory, session, args)
			return result, result.Error
		})

		if err != nil {
			if queueResult, isQueueResult := result.(queue.Result); isQueueResult {
				return queueResult
			}
			return queue.Failure(derp.Wrap(err, location, "Handler failed, did not return a queue.Result.  This should never happen."))
		}

		// Return the queue result
		if result, isQueueResult := result.(queue.Result); isQueueResult {
			return result
		}

		// Guard against panics if developers do bad things.  This should never happen.
		return queue.Failure(derp.InternalError(location, "Handler did not return a queue.Result.  This should never happen", result))
	})
}

// WithStream wraps a consumer function, using the "streamId" argument to load a Stream object from the database.
func WithStream(serverFactory ServerFactory, args mapof.Any, handler func(*service.Factory, data.Session, *service.Stream, *model.Stream, mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithStream"

	return WithSession(serverFactory, args, func(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByToken(session, args.GetString("streamId"), &stream); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load stream", args))
		}

		return handler(factory, session, streamService, &stream, args)
	})
}

// WithUser wraps a consumer function, using the "userId" argument to load a User object from the database.
func WithUser(serverFactory ServerFactory, args mapof.Any, handler func(*service.Factory, data.Session, *model.User, mapof.Any) queue.Result) queue.Result {

	const location = "consumer.WithUser"

	return WithSession(serverFactory, args, func(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByToken(session, args.GetString("userId"), &user); err != nil {
			return queue.Error(derp.Wrap(err, location, "Cannot load user", args))
		}

		return handler(factory, session, &user, args)
	})
}
