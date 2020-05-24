package routes

import (
	"github.com/benpate/data"
	"github.com/benpate/data/mockdb"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getTestDatastore() data.Datastore {

	return &mockdb.Datastore{
		"Stream": mockdb.Collection{
			&model.Stream{
				StreamID: primitive.NewObjectID(),
				Token:    "omg-it-works",
				Data: map[string]interface{}{
					"content": "this is my content.  deal with it.",
				},
			},
		},
	}
}
