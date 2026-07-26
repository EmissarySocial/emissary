package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version29 grants every existing User the new DIRECT_MESSAGE notification channel.
//
// User.NotificationChannels is a POSITIVE list -- User.NotificationEnabled returns TRUE only for a
// channel the slice actually contains. A newly-introduced channel is therefore absent for every
// existing User, and absent means suppressed: service.Notification.notify marks the notification
// born-read, so a direct message produces no unread dot, no SSE nudge, and no Web Push, and reports
// no error anywhere. Without this migration, adding the channel would silence direct messages for
// the entire existing user base.
//
// DECIDED: grant it to EVERY User, including those whose channel list is empty. An empty list is
// documented as the deliberate "everything off" state, so this does override that choice for those
// users -- accepted knowingly, on the grounds that a direct message is the one notification a user
// most likely wants even after muting the ambient ones. They can switch it back off in settings.
func Version29(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version29"

	fmt.Println("... Version 29")

	if err := addDirectMessageChannel(ctx, session); err != nil {
		return derp.Wrap(err, location, "Adding DIRECT_MESSAGE notification channel")
	}

	return nil
}

// addDirectMessageChannel appends the DIRECT_MESSAGE channel to every User's notificationChannels.
//
// $addToSet is chosen deliberately over $push: it is idempotent (a second run adds nothing), and it
// CREATES the array for the oldest records, which predate the field entirely and carry no
// notificationChannels key at all. No filter is applied for the same reason -- every User is a
// target, and the operation is a no-op for anyone who already has the channel.
func addDirectMessageChannel(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.addDirectMessageChannel"

	collection := session.Collection("User")

	update := bson.M{
		"$addToSet": bson.M{"notificationChannels": model.NotificationChannelDirectMessage},
	}

	result, err := collection.UpdateMany(ctx, bson.M{}, update)

	if err != nil {
		return derp.Wrap(err, location, "Updating User notification channels")
	}

	fmt.Println("...... granted DIRECT_MESSAGE to " + fmt.Sprint(result.ModifiedCount) + " users")

	return nil
}
