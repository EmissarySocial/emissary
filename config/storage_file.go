package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/benpate/derp"
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

	if args.Initialize {
		config := DefaultConfig()
		config.Source = storage.source
		config.Location = storage.location

		if err := storage.Write(config); err != nil {
			derp.Report(derp.Wrap(err, "config.FileStorage", "Error initializing MongoDB config"))
			panic("Error initializing File config: " + err.Error())
		}
	}

	// Listen for updates and post them to the update channel
	return storage
}

// Subscribe returns a channel that will receive the configuration every time it is updated
func (storage FileStorage) Subscribe() <-chan Config {

	updateChannel := make(chan Config, 1)

	go func() {

		fmt.Println("FileStorage: Loading configuration from:" + storage.location)
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
			fmt.Println("Refreshing configuration")
		}
	}()

	return updateChannel
}

// load reads the configuration from the filesystem and
// creates a default configuration if the file is missing
func (storage FileStorage) load() Config {

	result := NewConfig()

	// Try to load the configuration file from disk
	data, err := os.ReadFile(storage.location)

	if err != nil {
		// derp.Report(derp.Wrap(err, "config.FileStorage.load", "Error reading configuration"))

		fmt.Println("")
		fmt.Println("")
		fmt.Println("=====================================================")
		fmt.Println("EMISSARY: UNABLE TO READ CONFIGURATION...")
		fmt.Println("The configuration file is missing or invalid. Please")
		fmt.Println("check the following location:")
		fmt.Println("'" + storage.location + "'")
		fmt.Println("")
		fmt.Println("You can initialize an empty configuration file by")
		fmt.Println("running this command:")
		fmt.Println("> emissary --init --setup")
		fmt.Println("")
		fmt.Println("For assistance, please visit https://emissary.dev/")
		fmt.Println("=====================================================")
		fmt.Println("")
		fmt.Println("")

		os.Exit(1)
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
