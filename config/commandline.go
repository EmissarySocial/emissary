package config

import (
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

// CommandLineArgs represents the command line arguments passed to the server
type CommandLineArgs struct {
	Source     string // Type of configuration file (Command Line | Enviornment Variable | Default)
	Location   string // URI of the configuration file
	Database   string // Name of the MongoDB config database
	Collection string // Name of the MongoDB config collection
	Setup      bool   // If TRUE, then the server will run in SETUP mode
	HTTPPort   int    // Port to use in setup mode (only)
}

// GetCommandLineArgs returns the location of the configuration file
func GetCommandLineArgs() CommandLineArgs {

	var source string
	var location string
	var db string
	var collection string
	var setup bool
	var httpPort int

	// Look for the configuration location in the command line arguments
	pflag.StringVar(&location, "config", "", "Path to configuration file")
	pflag.StringVar(&db, "db", DefaultConfigDatabase, "Name of the MongoDB config database")
	pflag.StringVar(&collection, "collection", DefaultConfigCollection, "Name of the MongoDB config collection")
	pflag.BoolVar(&setup, "setup", false, "Run setup server")
	pflag.IntVar(&httpPort, "port", 0, "HTTP Port to use for setup mode.")
	pflag.Parse()

	// Whether --db was typed (as opposed to left at its default) decides whether a database name
	// in the connection string is allowed to win.  See resolveConfigDatabase.
	dbFromCommandLine := pflag.CommandLine.Changed("db")

	if location != "" {

		// Use command line argument for configuration
		log.Info().Msg("Locating configuration from command line argument.")
		source = ConfigSourceCommandLine

	} else if env := os.Getenv("EMISSARY_CONFIG"); env != "" {

		// Look for the configuration location in the environment
		log.Info().Msg("Locating configuration from environment variable.")
		source = ConfigSourceEnvironment
		location = env

		if envDb := os.Getenv("EMISSARY_CONFIG_DB"); envDb != "" {
			log.Info().Msg("ENV: Using configuration database: " + envDb)
			db = envDb
			dbFromCommandLine = true
		}

		if envCollection := os.Getenv("EMISSARY_CONFIG_COLLECTION"); envCollection != "" {
			log.Info().Msg("ENV: Using configuration collection: " + envCollection)
			collection = envCollection
		}

	} else {

		// Fall through to using default location (file in local directory)
		log.Info().Msg("No configuration specified. Using default location: `file://./config.json`")
		location = "file://./config.json"
		source = ConfigSourceDefault
	}

	return CommandLineArgs{
		Source:     source,
		Location:   location,
		Database:   resolveConfigDatabase(location, db, dbFromCommandLine),
		Collection: collection,
		Setup:      setup,
		HTTPPort:   httpPort,
	}
}

// resolveConfigDatabase decides which MongoDB database holds the server configuration.
//
// RULE: An explicit choice always wins, then the database named in the connection string, then
// the default.  The connection string leg is the one that used to be missing: `ConfigDatabase()`
// parsed it but nothing called it, so `mongodb://host/mycfgdb` silently read from "emissary"
// instead.  On a cluster whose nodes are configured inconsistently -- some with --db, some
// relying on the URI -- that put nodes on DIFFERENT collections, where they could never see each
// other's configuration changes and every node looked like it needed a reboot.
func resolveConfigDatabase(location string, database string, isExplicit bool) string {

	// An explicit --db flag or EMISSARY_CONFIG_DB overrides everything
	if isExplicit {
		return database
	}

	args := CommandLineArgs{Location: location}

	// A file:// config has no database at all
	if fromLocation := args.ConfigDatabase(); fromLocation != "" {
		return fromLocation
	}

	return database
}

// Protocol returns the storage protocol implied by the configuration location, or "" when the
// location matches no supported scheme.  Deciding what to DO about an unsupported scheme belongs
// to the caller -- Load turns it into an error, and main turns that into an exit.
func (args CommandLineArgs) Protocol() string {

	switch {

	case strings.HasPrefix(args.Location, "mongodb://"):
		return StorageTypeMongo

	case strings.HasPrefix(args.Location, "mongodb+srv://"):
		return StorageTypeMongo

	case strings.HasPrefix(args.Location, "file://"):
		return StorageTypeFile
	}

	return ""
}

// ConfigDatabase returns the MongoDB database named in the path of the configuration connection
// string, or the default when the connection string does not name one.  It returns "" for a
// file:// configuration, which has no database.
func (args CommandLineArgs) ConfigDatabase() string {

	if strings.HasPrefix(args.Location, "file://") {
		return ""
	}

	if location, err := url.Parse(args.Location); err == nil {
		location.Path = strings.TrimPrefix(location.Path, "/")
		if location.Path != "" {
			return location.Path
		}
	}

	return DefaultConfigDatabase
}

// ConfigOptions returns any config modifiers specified in the command line (like --port)
func (args CommandLineArgs) ConfigOptions() []Option {

	result := make([]Option, 0)

	if args.HTTPPort != 0 {
		result = append(result, WithHTTPPort(args.HTTPPort))
	}

	return result
}
