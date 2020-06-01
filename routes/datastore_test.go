package routes

import (
	"github.com/benpate/data"
	"github.com/benpate/data/mockdb"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getTestDatastore() data.Server {

	return &mockdb.Server{
		"Stream": mockdb.Collection{
			&model.Stream{
				StreamID: primitive.NewObjectID(),
				URL:      "http://localhost/omg-it-works",
				Token:    "omg-it-works",
				Title:    "OMG It Works",
				Summary:  "This is my first stream.  I can't believe it's working...",
				Data: map[string]interface{}{
					"content": "this is my content.  deal with it.",
				},
			},
		},
	}
}
