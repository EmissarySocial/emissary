package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
)

// Config defines all of the domains available on this server
type Config []Domain // Slice of one or more domain configurations

// Load retrieves all of the configured domains from permanent storage (currently filesystem)
func Load(filename string) (Config, error) {

	result := Default()

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return result, derp.Wrap(err, "ghost.config.Load", "Error loading config file", filename)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, derp.Wrap(err, "ghost.config.Load", "Error unmarshalling JSON", string(data))
	}

	return result, nil
}

// Default returns the default configuration for this application.
func Default() Config {
	return Config([]Domain{})
}

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config))

	for index := range config {
		result[index] = config[index].Hostname
	}

	return result
}
