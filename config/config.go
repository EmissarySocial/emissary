package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
)

type Global struct {
	Domains []Domain `json:"domains"` // Slice of one or more domain configurations
}

type Domain struct {
	Hostname      string   `json:"hostname"`            // Domain name of a virtual server
	ConnectString string   `json:"connectString"`       // MongoDB connect string
	DatabaseName  string   `json:"databaseName"`        // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	TemplatePaths []string `json:"templatePaths"`       // Paths to one or more directories where page templates are defined.
	ForwardTo     string   `json:"forwardTo,omitempty"` // Forwarding information for a domain that has moved servers
}

func Load(filename string) (Global, error) {

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

func Default() Global {

	return Global{
		Domains: []Domain{},
	}
}
