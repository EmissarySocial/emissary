package queries

import (
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func OAuthUserTokens(session data.Session, userID primitive.ObjectID) (sliceof.MapOfAny, error) {

	const location = "queries.OAuthUserToken"

	// Get the MongoDB Collection
	collection := mongoCollection(session.Collection("OAuthUserToken"))

	if collection == nil {
		return nil, derp.InternalError(location, "Unable to connect to OAuthUserToken collection")
	}

	// Build the Aggregate Pipeline
	pipeline := bson.A{

		// Limit to this UserID
		bson.M{"$match": bson.M{"userId": userID, "deleteDate": 0}},

		// Sort by create date (ascending)
		bson.M{"$sort": bson.M{"createDate": 1}},

		// LEFT JOIN OAuthClient
		bson.M{"$lookup": bson.M{
			"from":         "OAuthClient",
			"localField":   "clientId",
			"foreignField": "_id",
			"as":           "client",
		}},

		// Limit the values returned to the client
		bson.M{"$project": bson.M{
			"_id":              0,
			"oauthUserTokenId": "$_id",
			"clientId":         1,
			"createDate":       1,
			"scopes":           1,
			"actorId":          bson.M{"$arrayElemAt": bson.A{"$client.actorId", 0}},
			"name":             bson.M{"$arrayElemAt": bson.A{"$client.name", 0}},
			"iconUrl":          bson.M{"$arrayElemAt": bson.A{"$client.iconUrl", 0}},
			"website":          bson.M{"$arrayElemAt": bson.A{"$client.website", 0}},
			"summary":          bson.M{"$arrayElemAt": bson.A{"$client.summary", 0}},
		}},
	}

	// Query the database
	cursor, err := collection.Aggregate(session.Context(), pipeline)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to query database")
	}

	// Decode the query results
	result := sliceof.NewMapOfAny()

	if err := cursor.All(session.Context(), &result); err != nil {
		return nil, derp.Wrap(err, location, "Unable to decode cursor results")
	}

	// Success
	return result, nil
}
