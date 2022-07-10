package config

import (
	"context"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStorage is a MongoDB-backed configuration storage
type MongoStorage struct {
	Collection *mongo.Collection
}

// NewMongoStorage creates a fully initialized MongoStorage instance
func NewMongoStorage(location string) MongoStorage {

	// Try to make a new MongoDB connection
	client, err := mongo.NewClient(options.Client().ApplyURI(location))

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

	return MongoStorage{
		Collection: collection,
	}
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage MongoStorage) Subscribe() <-chan Config {

	result := make(chan Config, 1)
	ctx := context.Background()

	go func() {

		config, err := storage.load()

		if err != nil {
			derp.Report(derp.Wrap(err, "config.MongoStorage", "Error loading config from MongoDB"))
			panic("Error loading config from MongoDB: " + err.Error())
		}

		result <- config

		// watch for changes to the configuration
		cs, err := storage.Collection.Watch(ctx, mongo.Pipeline{})

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Watcher", "Unable to open Mongodb Change Stream"))
			return
		}

		for cs.Next(ctx) {
			if config, err := storage.load(); err == nil {
				result <- config
			}
		}
	}()

	return result
}

// load reads the configuration from the MongoDB database
func (storage MongoStorage) load() (Config, error) {

	result := NewConfig()

	if err := storage.Collection.FindOne(context.Background(), nil).Decode(&result); err != nil {
		return Config{}, derp.Wrap(err, "config.MongoStorage", "Error decoding config from MongoDB")
	}

	return result, nil
}
