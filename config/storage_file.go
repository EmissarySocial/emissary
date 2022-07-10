package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/fsnotify/fsnotify"
)

// FileStorage is a file-based storage engine for the server configuration
type FileStorage struct {
	location string
}

// NewFileStorage creates a fully initialized FileStorage instance
func NewFileStorage(location string) FileStorage {

	return FileStorage{
		location: location,
	}
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage FileStorage) Subscribe() <-chan Config {

	result := make(chan Config, 1) // Use a buffered channel to prevent blocking

	go func() {

		result <- storage.load()

		// Create a new file watcher
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		if err := watcher.Add(storage.location); err != nil {
			derp.Report(derp.Wrap(err, "Unable to watch for changes to configuration: ", storage.location))
			return
		}

		for {
			result <- storage.load()
			<-watcher.Events
		}
	}()

	return result
}

// load reads the configuration from the filesystem and
// creates a default configuration if the file is missing
func (storage FileStorage) load() Config {

	result := NewConfig()

	// Try to load the configuration file from disk
	data, err := ioutil.ReadFile(storage.location)

	if err != nil {
		derp.Report(derp.Wrap(err, "config.FileStorage.load", "Error reading config.json"))
		panic("Error reading configuration: " + storage.location)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		derp.Report(derp.Wrap(err, "config.FileStorage.load", "Error unmarshaling config.json"))
		panic("Invalid configuration file: " + err.Error())
	}

	return result
}

// Write writes the configuration to the filesystem
func (storage FileStorage) Write(config Config) error {

	// Marshal the configuration to JSON
	data, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error marshaling config.json")
	}

	// Try to write the configuration to disk
	if err := ioutil.WriteFile(storage.location, data, 0777); err != nil {
		return derp.Wrap(err, "config.FileStorage.Write", "Error writing config.json")
	}

	// Return nil if no errors were encountered
	return nil
}
