package config

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/fsnotify/fsnotify"
)

// FileStorage is a file-based storage engine for the server configuration
type FileStorage struct {
	source   string
	location string
}

// NewFileStorage creates a fully initialized FileStorage instance
func NewFileStorage(args CommandLineArgs) FileStorage {

	// Create a new FileStorage instance
	storage := FileStorage{
		source:   args.Source,
		location: strings.TrimPrefix(args.Location, "file://"),
	}

	// Listen for updates and post them to the update channel
	return storage
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage FileStorage) Subscribe() <-chan Config {

	updateChannel := make(chan Config, 1)

	go func() {

		spew.Dump("adding configuration")
		updateChannel <- storage.load()

		// Create a new file watcher
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		if err := watcher.Add(storage.location); err != nil {
			derp.Report(derp.Wrap(err, "Unable to watch for changes to configuration: ", storage.location))
			return
		}

		for range watcher.Events {
			updateChannel <- storage.load()
			spew.Dump("adding configuration")
		}
	}()

	return updateChannel
}

// load reads the configuration from the filesystem and
// creates a default configuration if the file is missing
func (storage FileStorage) load() Config {

	result := NewConfig()

	// Try to load the configuration file from disk
	data, err := ioutil.ReadFile(storage.location)

	if err != nil {
		derp.Report(derp.Wrap(err, "config.FileStorage.load", "Error reading configuration"))
		panic("Error reading configuration: " + storage.location)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		derp.Report(derp.Wrap(err, "config.FileStorage.load", "Error unmarshaling configuration"))
		panic("Invalid configuration file: " + err.Error())
	}

	result.Source = storage.source
	result.Location = storage.location

	return result
}

// Write writes the configuration to the filesystem
func (storage FileStorage) Write(config Config) error {

	// Marshal the configuration to JSON
	data, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error marshaling configuration")
	}

	// Try to write the configuration to disk
	if err := ioutil.WriteFile(storage.location, data, 0777); err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error writing configuration")
	}

	// Return nil if no errors were encountered
	return nil
}
