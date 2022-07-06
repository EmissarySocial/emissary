package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/pkg/browser"
	"github.com/sethvargo/go-password/password"
)

// CONFIG_FILENAME defines the relative path to the Whisperverse configuration file
const CONFIG_FILENAME = "./config.json"

func GetConfigLocation() string {

	if location := flag.String("config", "", "Path to configuration file"); location != nil {
		if *location != "" {
			return *location
		}
	}

	if location := os.Getenv("WHISPERVERSE_CONFIG"); location != "" {
		return location
	}

	return "file://./config.json"
}

// Load retrieves all of the configured domains from permanent storage (currently filesystem)
func Load() Config {

	location := GetConfigLocation()

	switch {
	case strings.HasPrefix(location, "file://"):
		return loadFromFile(strings.TrimPrefix(location, "file://"))

	case strings.HasPrefix(location, "mongodb://"):
		return loadFromMongoDB(location)

	case strings.HasPrefix(location, "mongodb+srv://"):
		return loadFromMongoDB(location)

	default:
		return createDefault()
	}
}

func loadFromFile(location string) Config {

	// Try to load the configuration file from disk
	data, err := ioutil.ReadFile(location)

	// If the file doesn't exist, create a default one
	if err != nil {
		return createDefault()
	}

	// Otherwise unmarshal and return
	result := NewConfig()
	if err := json.Unmarshal(data, &result); err != nil {
		panic("Invalid config.json: " + err.Error())
	}
	return result
}

func loadFromMongoDB(location string) Config {
	return Config{}
}

func createDefault() Config {

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
