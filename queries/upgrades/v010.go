package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version10 updates "AttributedTo" values to be single values, not slices
func Version10(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 10")

	err := ForEachRecord(session.Collection("Stream"), func(record mapof.Any) error {
		spew.Dump(record)
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

	return ForEachRecord(session.Collection("Inbox"), func(record mapof.Any) error {
		spew.Dump(record)
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
}
