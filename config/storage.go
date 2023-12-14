package config

import (
	"os"

	"github.com/rs/zerolog/log"
)

type Storage interface {
	Subscribe() <-chan Config
	Write(Config) error
}

// Load retrieves a Storage object from the location designated in the config file
func Load(args *CommandLineArgs) Storage {

	log.Debug().Msg("Loading server config from: " + args.Location)

	switch args.Protocol {

	case StorageTypeMongo:
		return NewMongoStorage(args)

	case StorageTypeFile:
		return NewFileStorage(args)
	}

	// Failure
	log.Error().Msg("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
	os.Exit(1)

	return nil
}
