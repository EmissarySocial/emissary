package config

import (
	"context"

	"github.com/benpate/derp"
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
}

// NewMongoStorage creates a fully initialized MongoStorage instance
func NewMongoStorage(args CommandLineArgs) MongoStorage {

	// Try to make a new MongoDB connection
	client, err := mongo.NewClient(options.Client().ApplyURI(args.Location))

	if err != nil {
		derp.Report(derp.Wrap(err, "config.NewMongoStorage", "Error creating MongoDB client"))
		panic("Error creating MongoDB client: " + err.Error())
	}

	// Try to connect to the MongoDB database
	if err := client.Connect(context.Background()); err != nil {
		derp.Report(derp.Wrap(err, "config.NewMongoStorage", "Error connecting to MongoDB"))
		panic("Error connecting to MongoDB: " + err.Error())
	}

	// Get the configuration collection
	collection := client.Database("emissary").Collection("config")

	storage := MongoStorage{
		source:        args.Source,
		location:      args.Location,
		collection:    collection,
		updateChannel: make(chan Config, 1),
	}

	if args.Initialize {

		// Delete existing configuration so we don't have multiple records.
		if _, err := storage.collection.DeleteMany(context.Background(), bson.M{}); err != nil {
			derp.Report(derp.Wrap(err, "config.MongoStorage", "Error deleting existing config"))
		}

		config := DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		if err := storage.Write(config); err != nil {
			derp.Report(derp.Wrap(err, "config.MongoStorage", "Error initializing MongoDB config"))
			panic("Error initializing MongoDB config: " + err.Error())
		}
	}

	// Listen for updates and post them to the update channel
	go func() {

		ctx := context.Background()

		config, err := storage.load()

		if err != nil {
			derp.Report(derp.Wrap(err, "config.MongoStorage", "Error loading config from MongoDB, and unable to write default configuration."))
			panic("Error loading config from MongoDB: " + err.Error())
		}

		storage.updateChannel <- config

		// watch for changes to the configuration
		cs, err := storage.collection.Watch(ctx, mongo.Pipeline{})

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Watcher", "Unable to open Mongodb Change Stream"))
			return
		}

		for cs.Next(ctx) {
			if config, err := storage.load(); err == nil {
				storage.updateChannel <- config
			}
		}
	}()

	return storage
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage MongoStorage) Subscribe() <-chan Config {
	return storage.updateChannel
}

// load reads the configuration from the MongoDB database
func (storage MongoStorage) load() (Config, error) {

	result := NewConfig()

	if err := storage.collection.FindOne(context.Background(), bson.M{}).Decode(&result); err != nil {
		return Config{}, derp.Wrap(err, "config.MongoStorage", "Error decoding config from MongoDB")
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
