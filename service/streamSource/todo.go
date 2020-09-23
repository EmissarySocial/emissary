package streamSource

import (
	"github.com/benpate/ghost/model"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TODO struct {
	SourceID primitive.ObjectID
}

func (todo TODO) Init(sourceID primitive.ObjectID, _ model.StreamSourceConfig) error {
	todo.SourceID = sourceID
	return nil
}

// JSONForm returns a JSON-Form object that collects the configuration data required for this adapter
func (todo TODO) JSONForm() string {
	return ""
}

func (todo TODO) Schema() schema.Schema {
	return schema.Schema{}
}

// Poll checks the remote data source and returnsa slice of model.Stream objects
func (todo TODO) Poll() ([]model.Stream, error) {
	return []model.Stream{}, nil
}

func (todo TODO) Webhook(data map[string]interface{}) (model.Stream, error) {
	return model.Stream{}, nil
}
