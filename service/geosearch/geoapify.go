package geosearch

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

func Geoapify(apiKey string) GeosearchFunc {

	const location = "service.geosearch.Geoapify"

	return func(query string) (sliceof.Object[Place], error) {

		response := mapof.NewAny()

		txn := remote.Get("https://api.geoapify.com/v1/geocode/autocomplete").
			Query("text", query).
			Query("apiKey", apiKey).
			Result(&response)

		if err := txn.Send(); err != nil {
			return nil, derp.Wrap(err, location, "Unable to retrieve search results")
		}

		features := response.GetSliceOfMap("features")

		result := make(sliceof.Object[Place], 0, len(features))

		for _, feature := range features {

			props := feature.GetMap("properties")

			place := Place{
				Name:      first.String(props.GetString("name"), props.GetString("formatted")),
				Latitude:  props.GetFloat("lat"),
				Longitude: props.GetFloat("lon"),
			}

			result = append(result, place)
		}

		return result, nil
	}
}
