package config

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/fsnotify/fsnotify"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// FileStorage is a file-based storage engine for the server configuration
type FileStorage struct {
	source        string
	location      string
	updateChannel chan Config
	closeChannel  chan struct{}
}

// NewFileStorage creates a fully initialized FileStorage instance
func NewFileStorage(args *CommandLineArgs) FileStorage {

	fileLocation := strings.TrimPrefix(args.Location, "file://")

	// Create a new FileStorage instance
	storage := FileStorage{
		source:        args.Source,
		location:      fileLocation,
		updateChannel: make(chan Config, 1),
		closeChannel:  make(chan struct{}),
	}

	// Special rules for the first time we load the configuration file
	config, err := storage.load()

	switch {

	// If the config was read successfully, then NOOP here skips down to the next section.
	case err == nil:

	// If the config was not found, then run in setup mode and create a new default configuration
	case derp.NotFound(err):

		if !args.Setup {
			log.Error().Msg("Emissary could not start because the configuration file could not be found.")
			log.Error().Msg("Please re-run Emissary with the --setup flag to create a new configuration file.")
			os.Exit(1)
		}

		log.Debug().Msg("Configuration file not found.  Running in setup mode.")

		// Overwrite the configuration with a default value
		config = DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		// Save the config to disk
		if err := storage.Write(config); err != nil {
			derp.Report(derp.Wrap(err, "config.FileStorage", "Error initializing MongoDB config"))
			log.Error().Msg("Error initializing File config: " + err.Error())
			os.Exit(1)
		}

	// Anything but a "Not Found" error is a catastrophic failure.
	default:
		derp.Report(err)
		log.Error().Msg("FATAL: Emissary could not start because the configuration file could not be read.")
		log.Error().Msg("Check the file in location: " + fileLocation)
		os.Exit(1)
	}

	// If we have a valid config, post it to the update channel
	storage.updateChannel <- config

	// After the first load, watch for changes to the configuration file and post them to the update channel
	go func() {

		// Create a new file watcher
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		defer watcher.Close()

		if err := watcher.Add(storage.location); err != nil {
			derp.Report(derp.Wrap(err, "Unable to watch for changes to configuration: ", fileLocation))
			return
		}

		for {
			select {

			case <-storage.closeChannel:
				return

			case <-watcher.Events:
				if config, err := storage.load(); err == nil {
					if config.IsEmpty() {
						continue
					}
					storage.updateChannel <- config
				} else {
					derp.Report(derp.Wrap(err, "config.FileStorage", "Error loading the updated config from ", fileLocation))
				}

			case err := <-watcher.Errors:
				derp.Report(derp.Wrap(err, "config.FileStorage", "Error watching for changes to ", fileLocation))
			}
		}

	}()

	// Listen for updates and post them to the update channel
	return storage
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage FileStorage) Subscribe() <-chan Config {
	return storage.updateChannel
}

func (storage FileStorage) Close() {
	close(storage.closeChannel)
}

// load reads the configuration from the filesystem and
// creates a default configuration if the file is missing
func (storage FileStorage) load() (Config, error) {

	result := NewConfig()

	// Try to load the configuration file from disk
	data, err := os.ReadFile(storage.location)

	if err != nil {
		return Config{}, derp.Wrap(err, "config.FileStorage.load", "Error reading configuration", derp.WithNotFound())
	}

	if err := hjson.Unmarshal(data, &result); err != nil {
		return Config{}, derp.InternalError("config.FileStorage.load", "Error unmarshaling configuration", derp.WithWrappedValue(err))
	}

	result.Source = storage.source
	result.Location = storage.location

	return result, nil
}

// Write writes the configuration to the filesystem
func (storage FileStorage) Write(config Config) error {

	// Marshal the configuration to JSON
	data, err := json.MarshalIndent(config, "", "    ")

	if err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error marshaling configuration")
	}

	// Try to write the configuration to disk
	if err := os.WriteFile(storage.location, data, 0777); err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error writing configuration")
	}

	// Return nil if no errors were encountered
	return nil
}
