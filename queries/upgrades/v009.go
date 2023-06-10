package upgrades

import (
	"context"
	"fmt"
	"strings"

	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version9 moves all `journal.*` fields into the top level of each model object
func Version9(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 8")

	ForEachRecord(session.Collection("Stream"), func(record mapof.Any) error {
		if content, ok := record["content"]; ok {
			if contentMap, ok := content.(mapof.Any); ok {
				contentMap["contentType"] = strings.ToUpper(contentMap.GetString("type"))
			}
		}
		return nil
	})

	return nil
}
