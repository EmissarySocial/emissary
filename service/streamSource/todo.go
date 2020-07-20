package streamSource

import (
	"github.com/benpate/ghost/model"
	"github.com/qri-io/jsonschema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TODO struct {
	SourceID primitive.ObjectID
}

func (todo TODO) Init(sourceID primitive.ObjectID, _ model.StreamSourceConfig) error {
	todo.SourceID = sourceID
	return nil
}

// JSONSchema returns a JSON-Schema object that can validate the configuration data required for this adapter
func (todo TODO) JSONSchema() jsonschema.Schema {
	return jsonschema.Schema{}
}

// JSONForm returns a JSON-Form object that collects the configuration data required for this adapter
func (todo TODO) JSONForm() string {
	return ""
}

// Poll checks the remote data source and returnsa slice of model.Stream objects
func (todo TODO) Poll() ([]model.Stream, error) {
	return []model.Stream{}, nil
}

func (todo TODO) Webhook(data map[string]interface{}) (model.Stream, error) {
	return model.Stream{}, nil
}
