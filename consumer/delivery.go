package consumer

import (
	"context"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// Delivery runs one task of hannibal's delivery pipeline (a fan-out or a single send) for the
// domain of the actor that sent the activity
func Delivery(serverFactory ServerFactory, task queue.Task) queue.Result {

	return WithFactory(serverFactory, task.Arguments, func(factory *service.Factory, args mapof.Any) queue.Result {

		const location = "consumer.Delivery"

		// RULE: A plain session, deliberately NOT a transaction. A send makes an HTTP request, and
		// a fan-out can outlast MongoDB's limit on a transaction. Neither writes to this database.
		session, err := factory.Server().Session(context.Background())

		if err != nil {
			return queue.Error(derp.Wrap(err, location, "Opening database session"))
		}

		defer session.Close()

		// The pipeline is built for this one task, bound to the actor that the task names
		pipeline := factory.Outbox2().Pipeline(session, args.GetString("actor"), serverFactory.AllowPrivateIPs())

		return delivery.Consumer(pipeline).Run(task)
	})
}

// Outbox2Email emails one activity to one recipient
func Outbox2Email(factory *service.Factory, args mapof.Any) queue.Result {

	const location = "consumer.Outbox2Email"

	// RULE: A plain session, deliberately NOT a transaction, because sending email is a network call
	session, err := factory.Server().Session(context.Background())

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Opening database session"))
	}

	defer session.Close()

	if err := factory.Outbox2().SendEmail(session, args.GetString("actor"), args.GetString("address"), args.GetMap("activity")); err != nil {
		return queue.Error(derp.Wrap(err, location, "Sending email", "activityId: "+args.GetString("activityId")))
	}

	return queue.Success()
}
