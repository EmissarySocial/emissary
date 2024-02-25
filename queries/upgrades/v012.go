package upgrades

import (
	"context"
	"fmt"

	"github.com/EmissarySocial/emissary/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version12 updates "AttributedTo" values to be single values, not slices
func Version12(ctx context.Context, session *mongo.Database) error {

	collection := session.Collection("Stream")

	criteria := bson.M{"deleteDate": 0}
	cursor, err := collection.Find(ctx, criteria)

	if err != nil {
		return err
	}

	streams := make([]model.Stream, 0)
	if err := cursor.All(ctx, &streams); err != nil {
		return err
	}

	for _, stream := range streams {

		filter := bson.M{"parentId": stream.StreamID}
		update := bson.M{"$set": bson.M{"parentTemplateId": stream.TemplateID}}

		if _, err := collection.UpdateMany(ctx, filter, update); err != nil {
			return err
		}
	}

	fmt.Println("... Version 12")
	return nil
}
