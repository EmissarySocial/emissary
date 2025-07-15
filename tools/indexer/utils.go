package indexer

import (
	"context"
	"iter"
	"reflect"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func rangeFunc[T any](ctx context.Context, cursor *mongo.Cursor) iter.Seq[T] {

	const location = "tools.indexer.rangeFunc"

	return func(yield func(value T) bool) {

		for cursor.Next(ctx) {

			select {
			case <-ctx.Done():
				return

			default:

				var value T
				if err := cursor.Decode(&value); err != nil {
					derp.Report(derp.Wrap(err, location, "Error decoding record"))
					continue
				}

				if !yield(value) {
					return
				}
			}
		}
	}
}

func compareModel(currentIndex mapof.Any, newIndex mongo.IndexModel) bool {

	// Without options, this cannot be a match.
	if newIndex.Options == nil {
		log.Trace().Msg("compareModel: No options provided for new index")
		return false
	}

	// Name must match
	if currentIndex.GetString("name") != *newIndex.Options.Name {
		log.Trace().Msg("compareModel: Index names do not match")
		return false
	}

	// Compare UNIQUE option
	if unique := newIndex.Options.Unique; unique != nil {
		if currentIndex.GetBool("unique") != *unique {
			log.Trace().Msg("compareModel: Index UNIQUE options do not match")
			return false
		}
	} else if _, exists := currentIndex["unique"]; exists {
		log.Trace().Msg("compareModel: Index UNIQUE options do not match")
		return false
	}

	// Compare SPARSE option
	if sparse := newIndex.Options.Sparse; sparse != nil {
		if currentIndex.GetBool("sparse") != *sparse {
			log.Trace().Msg("compareModel: Index SPARSE options do not match")
			return false
		}
	} else if _, exists := currentIndex["sparse"]; exists {
		log.Trace().Msg("compareModel: Index SPARSE options do not match")
		return false
	}

	// Compare KEYS by converting the new index into a bson.D
	currentKeys := currentIndex.GetSliceOfMap("key")
	newKeys := normalizeNewKeys(newIndex)

	if !reflect.DeepEqual(currentKeys, newKeys) {
		log.Trace().Msg("compareModel: Index KEYS do not match")
		return false
	}

	// If we reach here, the indexes match
	log.Trace().Msg("compareModel: Indexes match")
	return true
}

func normalizeNewKeys(index mongo.IndexModel) []mapof.Any {

	// Convert single items bson.E into a slice of maps
	if singleItem, isSingleItem := index.Keys.(bson.E); isSingleItem {
		return []mapof.Any{
			{singleItem.Key: int32(convert.Int(singleItem.Value))},
		}
	}

	// Convert multi-item bson.D into a slice of maps
	if multiItem, isMultiItem := index.Keys.(bson.D); isMultiItem {
		result := make([]mapof.Any, len(multiItem))

		for index, item := range multiItem {
			result[index] = mapof.Any{
				item.Key: int32(convert.Int(item.Value)),
			}
		}
		return result
	}

	// Unknown conversion.  This should never happen.
	return []mapof.Any{}
}
