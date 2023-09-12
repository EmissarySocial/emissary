package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version11 updates "AttributedTo" values to be single values, not slices
func Version11(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 11")

	for _, collection := range []string{"Stream", "StreamSummary", "Inbox"} {
		err := ForEachRecord(session.Collection(collection), func(record mapof.Any) error {
			if attributedTo, ok := record["attributedTo"]; ok {
				if attributedToSlice, ok := attributedTo.([]any); ok {
					if len(attributedToSlice) > 0 {
						record["attributedTo"] = attributedToSlice[0]
						return nil
					}
				}
				record["attributedTo"] = model.NewPersonLink()
			}
			return nil
		})

		if err != nil {
			return err
		}

	}

	return nil
}
