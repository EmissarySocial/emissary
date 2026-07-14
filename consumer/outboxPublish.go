package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxPublish is the post-commit fan-out for a published activity. Outbox.Publish saves the
// OutboxMessage and mints the activity ID inside the request transaction, then enqueues this task,
// which runs only AFTER that transaction commits — so no signed HTTP delivery ever happens inside
// an open transaction (POST-COMMIT-FEDERATION.md F2). The filtering + dispatch lives on the Outbox
// service (Outbox.Deliver) so it keeps its internal service access; this consumer only unmarshals args.
func OutboxPublish(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.OutboxPublish"

	// Parse the sending Actor
	actorType := args.GetString("actorType")

	actorID, err := primitive.ObjectIDFromHex(args.GetString("actorId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid 'actorId' argument", args))
	}

	// The activity payload already carries its minted `id` and authoritative `actor` (set by Publish).
	activity := args.GetMap("activity")

	if len(activity) == 0 {
		return queue.Failure(derp.Internal(location, "Missing 'activity' argument", args))
	}

	// Re-parse permissions (serialized as hex strings — ObjectIDs don't survive task storage).
	permissions := make(model.Permissions, 0)

	for _, permissionHex := range convert.SliceOfString(args.GetAny("permissions")) {
		if permissionID, err := primitive.ObjectIDFromHex(permissionHex); err == nil {
			permissions = append(permissions, permissionID)
		}
	}

	// The WithRecipients override (author-only delivery). hasRecipients distinguishes "no override"
	// from "empty override"; both must survive the round-trip or reactions fan out to all followers.
	recipients := convert.SliceOfString(args.GetAny("recipients"))
	hasRecipients := args.GetBool("hasRecipients")

	// Fan out. Per-recipient failures are logged inside Deliver and do not fail the task; the
	// actual deliveries are independently retryable OutboxSendToSingleRecipient sub-tasks.
	if err := factory.Outbox().Deliver(session, actorType, actorID, activity, permissions, recipients, hasRecipients); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to fan out activity", args))
	}

	return queue.Success()
}
