package config

import (
	"context"
	"os"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStorage is a MongoDB-backed configuration storage
type MongoStorage struct {
	source        string
	location      string
	collection    *mongo.Collection
	updateChannel chan Config
	cancelFunc    context.CancelFunc
}

// NewMongoStorage creates a fully initialized MongoStorage instance
func NewMongoStorage(args *CommandLineArgs) MongoStorage {

	const location = "config.NewMongoStorage"

	// Create a new MongoDB database connection
	connectOptions := options.Client().ApplyURI(args.Location)
	client, err := mongo.Connect(context.Background(), connectOptions)

	if err != nil {
		log.Error().Msg("Emissary cannot start because the MongoDB config database could not be reached.")
		log.Error().Msg("Check the MongoDB connection string and verify the database server connection.")
		log.Error().Err(err).Send()
		os.Exit(1)
	}

	// Get the configuration collection
	collection := client.Database("emissary").Collection("config")

	context, cancelFunc := context.WithCancel(context.Background())

	storage := MongoStorage{
		source:        args.Source,
		location:      args.Location,
		collection:    collection,
		updateChannel: make(chan Config, 1),
		cancelFunc:    cancelFunc,
	}

	// Special rules for the first time we load the configuration file
	config, err := storage.load()

	switch {

	// If the config was read successfully, then NOOP here skips down to the next section.
	case err == nil:

	case derp.IsNotFound(err):

		if !args.Setup {
			log.Error().Msg("Emissary could not start because the configuration database could not be found.")
			log.Error().Msg("Please re-run Emissary with the --setup flag to initialize the configuration database.")
			os.Exit(1)
		}

		// Create a default configuration
		config = DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		if inner := storage.Write(config); inner != nil {
			log.Error().Msg("Error writing new configuration file to the Mongo database")
			log.Error().Err(inner).Send()
			os.Exit(1)
		}

	default:

		derp.Report(err)

		// Any other errors connecting to the Mongo server will prevent Emissary from starting.
		log.Error().Msg("Emissary could not start because of an error connecting to the MongoDB config database.")
		log.Error().Err(err).Send()
		os.Exit(1)
	}

	// If we have a valid config, post it to the update channel
	storage.updateChannel <- config

	log.Info().Msgf("Loading configuration from mongodb")

	// After the first load, watch for changes to the config record and post them to the update channel
	go func() {

		// watch for changes to the configuration
		cs, err := storage.collection.Watch(context, mongo.Pipeline{})

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to open Mongodb Change Stream"))
			return
		}

		for cs.Next(context) {

			if config, err := storage.load(); err == nil {
				storage.updateChannel <- config
			} else {
				derp.Report(derp.Wrap(err, location, "Unable to load updated config from MongoDB"))
			}
		}

		if err := cs.Err(); err != nil {
			derp.Report(derp.Wrap(err, location, "Error watching Mongodb Change Stream"))
		}
	}()

	return storage
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage MongoStorage) Subscribe() <-chan Config {
	return storage.updateChannel
}

func (storage MongoStorage) Close() {
	storage.cancelFunc()
}

// load reads the configuration from the MongoDB database
func (storage MongoStorage) load() (Config, error) {

	result := NewConfig()

	if err := storage.collection.FindOne(context.Background(), bson.M{}).Decode(&result); err != nil {

		if err == mongo.ErrNoDocuments {
			return Config{}, derp.NotFound("config.MongoStorage", "Unable to load config from MongoDB", err.Error())
		}

		return Config{}, derp.Wrap(err, "config.MongoStorage", "Unable to decode config from MongoDB")
	}

	result.Source = storage.source
	result.Location = storage.location

	return result, nil
}

// Write writes the configuration to the database
func (storage MongoStorage) Write(config Config) error {

	upsert := true
	criteria := bson.M{"_id": config.MongoID}
	options := options.ReplaceOptions{
		Upsert: &upsert,
	}

	if _, err := storage.collection.ReplaceOne(context.Background(), criteria, config, &options); err != nil {
		return derp.Wrap(err, "config.MongoStorage", "Error writing config to MongoDB")
	}

	return nil
}
