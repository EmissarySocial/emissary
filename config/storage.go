package config

import (
	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// Storage reads the server Config from its home (a file or a database), and publishes every update
type Storage interface {

	// Subscribe returns a channel that receives the configuration every time it changes
	Subscribe() <-chan Config

	// Read returns the configuration as currently stored.  Callers use it to rebase a
	// read-modify-write after Write reports a revision conflict.
	Read() (Config, error)

	// Write persists the configuration and returns it AS STORED -- with its Revision
	// incremented.  RULE: Write is a compare-and-swap on Config.Revision.  A value whose
	// Revision no longer matches the stored document returns a 409 (derp.IsConflict) and
	// changes NOTHING: overwriting blindly would silently destroy whatever another node
	// changed since this value was read -- up to and including a domain's MasterKey, which
	// lives nowhere else.  Callers keep the returned value: a later save based on the input
	// (with its stale Revision) would conflict against its own write.
	Write(Config) (Config, error)

	// Close shuts down the storage engine's watcher
	Close()
}

// Load builds the Storage engine designated by the configuration location.
func Load(args *CommandLineArgs) (Storage, error) {

	// RULE: This package never decides to end the process.  Every failure -- an unsupported scheme,
	// an unreachable database, a missing file without --setup -- comes back as an error carrying the
	// operator-facing guidance, and main decides what to do with it.  That is what makes every
	// failure branch in the constructors testable.

	const location = "config.Load"

	switch args.Protocol() {

	case StorageTypeMongo:
		log.Info().Msg("Loading server config from MongoDB ")
		return NewMongoStorage(args)

	case StorageTypeFile:
		log.Info().Msg("Loading server config from file: " + args.Location)
		return NewFileStorage(args)
	}

	return nil, derp.Internal(location, "Invalid configuration location. Must be file:// or mongodb:// or mongodb+srv://", args.Location)
}
