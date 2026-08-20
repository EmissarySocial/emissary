package config

import (
	"os"

	"github.com/rs/zerolog/log"
)

// Storage reads the server Config from its home (a file or a database), and publishes every update
type Storage interface {
	Subscribe() <-chan Config
	Write(Config) error
	Close()
}

// Load retrieves a Storage object from the location designated in the config file
func Load(args *CommandLineArgs) Storage {

	switch args.Protocol() {

	case StorageTypeMongo:
		log.Info().Msg("Loading server config from MongoDB ")
		return NewMongoStorage(args)

	case StorageTypeFile:
		log.Info().Msg("Loading server config from file: " + args.Location)
		return NewFileStorage(args)
	}

	// Failure
	log.Error().Msg("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
	os.Exit(1)

	return nil
}
