package indexer

import (
	"context"
	"iter"
	"reflect"

	"github.com/benpate/derp"
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

// compareModel returns TRUE if the index and model are identical
func compareModel(currentIndex mapof.Any, newIndex mongo.IndexModel) bool {

	newIndexMap := convertModelToMap(newIndex)

	if reflect.DeepEqual(currentIndex, newIndexMap) {
		return true
	}

	log.Trace().Msg("indexes do not match")
	// spew.Dump(newIndex, newIndexMap, currentIndex)
	return false
}

func convertModelToMap(newIndex mongo.IndexModel) mapof.Any {

	result := mapof.NewAny()

	if keys := newIndex.Keys; keys != nil {
		result["key"] = primitiveToMap(keys)
	}

	if newIndex.Options == nil {
		log.Trace().Msg("convertModelToMap: No options provided for new index")
		return result
	}

	if name := newIndex.Options.Name; name != nil {
		result["name"] = *name
	}

	if unique := newIndex.Options.Unique; unique != nil {
		result["unique"] = *unique
	}

	if sparse := newIndex.Options.Sparse; sparse != nil {
		result["sparse"] = *sparse
	}

	if version := newIndex.Options.Version; version != nil {
		result["v"] = int32(*version)
	} else {
		result["v"] = int32(2)
	}

	if sphereIndexVersion := newIndex.Options.SphereVersion; sphereIndexVersion != nil {
		result["2dsphereIndexVersion"] = int32(*sphereIndexVersion)
	}

	if weights := newIndex.Options.Weights; weights != nil {
		result["weights"] = primitiveToMap(weights)

		// Set the default text index version for full-text indexes
		if textVersion := newIndex.Options.TextVersion; textVersion != nil {
			result["textIndexVersion"] = int32(*textVersion)
		} else {
			result["textIndexVersion"] = int32(3)
		}

		// Set the default language override for full-text indexes
		if language := newIndex.Options.LanguageOverride; language != nil {
			result["language_override"] = *language
		} else {
			result["language_override"] = "language"
		}

		// Set the default language for full-text indexes
		if language := newIndex.Options.DefaultLanguage; language != nil {
			result["default_language"] = *language
		} else {
			result["default_language"] = "english"
		}
	}

	if partialFilter := newIndex.Options.PartialFilterExpression; partialFilter != nil {
		result["partialFilterExpression"] = primitiveToMap(partialFilter)
	}

	return result
}

func primitiveToMap(input any) mapof.Any {

	result := mapof.NewAny()

	// Convert bson.E into mapof.Any
	if primitiveE, isPrimitiveE := input.(bson.E); isPrimitiveE {
		result[primitiveE.Key] = convertMapValue(primitiveE.Value)
		return result
	}

	// Convert bson.D into a mapof.Any
	if primitiveD, isPrimitiveD := input.(bson.D); isPrimitiveD {
		result := mapof.NewAny()

		for _, primitiveE := range primitiveD {
			result[primitiveE.Key] = convertMapValue(primitiveE.Value)
		}

		return result
	}

	// Convert bson.M into a mapof.Any
	if primitiveM, isPrimitiveM := input.(bson.M); isPrimitiveM {

		for key, value := range primitiveM {
			result[key] = convertMapValue(value)
		}

		return result
	}

	log.Debug().Msg("primitiveToMap: Unrecognized input.. returning empty map.")
	return result
}

func convertMapValue(value any) any {

	switch typedValue := value.(type) {

	case int:
		return int32(typedValue)
	case string:
		return typedValue
	case bool:
		return typedValue
	case bson.M:
		return primitiveToMap(typedValue)
	}

	// Fallback for other types
	log.Debug().Msg("Unrecognized type in primitive map: " + reflect.TypeOf(value).String())

	return value
}
