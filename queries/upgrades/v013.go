package upgrades

import (
	"context"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version13 updates User's InboxTemplate and OutboxTemplate values
func Version13(ctx context.Context, session *mongo.Database) error {

	return ForEachRecord(session.Collection("User"), func(record mapof.Any) error {
		if _, ok := record["inboxTemplate"]; !ok {
			record["inboxTemplate"] = "user-inbox"
		}

		if _, ok := record["outboxTemplate"]; !ok {
			record["outboxTemplate"] = "user-outbox"
		}

		return nil
	})
}
