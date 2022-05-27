package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/benpate/derp"
	"github.com/pkg/browser"
	"github.com/sethvargo/go-password/password"
)

// CONFIG_FILENAME defines the relative path to the Whisperverse configuration file
const CONFIG_FILENAME = "./config.json"

// Load retrieves all of the configured domains from permanent storage (currently filesystem)
func Load() Config {

	// Try to load the configuration file from disk
	{
		data, err := ioutil.ReadFile(CONFIG_FILENAME)

		// If successful, then unmarshal and return
		if err == nil {
			result := NewConfig()
			if err := json.Unmarshal(data, &result); err != nil {
				panic("Invalid config.json: " + err.Error())
			}
			return result
		}

		fmt.Println("Error loading configuration file:" + err.Error())
	}

	// Fall through means it's the first run, and we're generating a new default config from scratch.
	{
		fmt.Println("Generating new file...")

		defaultAdminURL, err := password.Generate(36, 10, 0, false, false)

		if err != nil {
			panic("Error generating default admin location: " + err.Error())
		}

		// Create a default password
		defaultPassword, err := password.Generate(36, 10, 0, false, false)

		if err != nil {
			panic("Error generating default password: " + err.Error())
		}

		// Generate a new configuration file using the default password
		result := DefaultConfig(defaultAdminURL, defaultPassword)
		data, err := json.MarshalIndent(result, "", "\t")

		if err != nil {
			panic("Error marshaling new config file: " + err.Error())
		}

		// Try to write the new configuration to the file system
		if err := ioutil.WriteFile(CONFIG_FILENAME, data, 0777); err != nil {
			panic("Error writing default configuration: " + err.Error() + "\n Check file permissions.")
		}

		// Output helpful hints for system admins
		fmt.Println("Default config file created at: " + CONFIG_FILENAME)
		fmt.Println("")
		fmt.Println("SAVE THIS INFORMATION SOMEWHERE SAFE")
		fmt.Println("-----------------------------------------")
		fmt.Println("To access the server admin, and configure")
		fmt.Println("websites on this server open this URL:")
		fmt.Println("http://localhost/" + result.AdminURL)
		fmt.Println("The admin password is: " + result.AdminPassword)
		fmt.Println("-----------------------------------------")

		// On first run, open web browser in admin mode
		browser.OpenURL("http://localhost/" + result.AdminURL + "?first=true")

		// Success!
		return result
	}
}

// Write saves the current configuration to permanent storage (currently filesystem)
func Write(config Config, filename string) error {

	output, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		return derp.Wrap(err, "config.Write", "Error marshalling configuration")
	}

	if err := os.WriteFile(filename, output, 0x777); err != nil {
		return derp.Wrap(err, "config.Write", "Error writing configuration")
	}

	return nil
}
