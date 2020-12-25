package streamSource

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Source defines the interface for all "source" adapters, that know how to connect to a (likely remote) data source and
// generate a slice of model.Stream objects to be imported into the local database.
type StreamSource interface {
	Init(primitive.ObjectID, model.StreamSourceConfig) error
	JSONForm() string
	Schema() schema.Schema

	Poll() ([]model.Stream, error)
	Webhook(map[string]interface{}) (model.Stream, error)
}

// New uses a map of configuration information to return a fully populated Source of model.Stream objects (almost certainly from a remote server)
func New(adapter model.StreamSourceAdapter, sourceID primitive.ObjectID, config model.StreamSourceConfig) (StreamSource, error) {

	var result StreamSource

	switch adapter {

	case model.StreamSourceAdapterActivityPub:
		result = TODO{}

	case model.StreamSourceAdapterEmail:
		result = TODO{}

	case model.StreamSourceAdapterRSS:
		result = &RSS{}

	case model.StreamSourceAdapterSystem:
		result = TODO{}

	case model.StreamSourceAdapterTwitter:
		result = TODO{}

	default:
		return nil, derp.New(500, "service.NewSource", "Unrecognized Stream Source", config["type"])
	}

	// Get the Schema, and validate the configuration data against it.
	schema := result.Schema()

	if err := schema.Validate(config); err != nil {
		return result, derp.Wrap(err, "source.NewSource", "Invalid configuration info", config)
	}

	// Pass the configuration data into the source adapter
	if err := result.Init(sourceID, config); err != nil {
		return result, derp.Wrap(err, "source.NewSource", "Error populating Source with provided configuration data")
	}

	return result, nil
}
