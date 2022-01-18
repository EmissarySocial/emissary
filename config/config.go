package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/benpate/derp"
)

// Config defines all of the domains available on this server
type Config struct {
	StorageType     string   `json:"storageType"`    // How this file is stored (currently, only "FILE")
	StorageLocation string   `json:"stoageLocation"` // Where the file is stored (currently, only the file path)
	Password        string   `json:"password"`       // Password for access to admin
	Domains         []Domain `json:"domains"`        // Slice of one or more domain configurations
}

// Load retrieves all of the configured domains from permanent storage (currently filesystem)
func Load(filename string) (Config, error) {

	result := Default()

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return result, derp.Wrap(err, "whisper.config.Load", "Error loading config file", filename)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, derp.Wrap(err, "whisper.config.Load", "Error unmarshalling JSON", string(data))
	}

	return result, nil
}

// Write saves the current configuration to permanent storage (currently filesystem)
func Write(config Config, filename string) error {

	output, err := json.MarshalIndent(config, "\n", "\t")

	if err != nil {
		return derp.Wrap(err, "whisper.config.Write", "Error marshalling configuration")
	}

	if err := os.WriteFile(filename, output, 0x777); err != nil {
		return derp.Wrap(err, "whisper.config.Write", "Error writing configuration")
	}

	return nil
}

// Default returns the default configuration for this application.
func Default() Config {
	return Config{
		StorageType:     "FILE",
		StorageLocation: "./config.json",
		Domains:         make([]Domain, 0),
	}
}

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config.Domains))

	for index := range config.Domains {
		result[index] = config.Domains[index].Hostname
	}

	return result
}
