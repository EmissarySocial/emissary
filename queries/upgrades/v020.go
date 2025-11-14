package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/geo"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/mongo"
)

// Version20...
func Version20(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 20")

	return ForEachRecord(session.Collection("Stream"), func(record mapof.Any) bool {

		var places sliceof.Object[mapof.Any] = record.GetSliceOfMap("places")

		if places.IsZero() {
			return false
		}

		place := places.First()

		record["location"] = geo.Address{
			Formatted: place.GetString("fullAddress"),
			Street1:   place.GetString("street1"),
			Street2:   place.GetString("street2"),
			Locality:  place.GetString("locality"),
			Region:    place.GetString("region"),
			Country:   place.GetString("country"),
			Longitude: place.GetFloat("longitude"),
			Latitude:  place.GetFloat("latitude"),
		}

		return true
	})
}
