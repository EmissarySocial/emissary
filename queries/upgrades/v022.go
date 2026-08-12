package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version22...
func Version22(ctx context.Context, session *mongo.Database) error {

	const location = "upgrade.Version22"

	fmt.Println("... Version 22")

	inbox := session.Collection("Inbox")
	newsFeed := session.Collection("NewsFeed")

	// Try to move all records from the Inbox to the NewsFeed
	if err := ForEachRecord(inbox, func(record mapof.Any) bool {
		const location = "upgrade.Version22"

		// Save the record to the NewsFeed
		if _, err := newsFeed.InsertOne(ctx, record); err != nil {
			fmt.Printf(location+": Unable to copy Inbox record to NewsFeed: %v\n", err)
			// DO NOT FAIL THE ENTIRE UPGRADE IF ONE RECORD FAILS TO COPY. Just skip it and move on to the next one.
			// derp.Report(derp.Wrap(err, location, "Copying Inbox record to NewsFeed", record["_id"]))
			// return false
		}

		return true
	}); err != nil {
		return derp.Wrap(err, location, "Copying records to NewsFeed")
	}

	// Drop the Inbox
	if err := inbox.Drop(ctx); err != nil {
		return derp.Wrap(err, location, "Dropping Inbox collection")
	}

	return nil
}
