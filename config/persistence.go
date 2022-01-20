package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/benpate/derp"
)

// Load retrieves all of the configured domains from permanent storage (currently filesystem)
func Load(filename string) (Config, error) {

	result := NewConfig()

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
