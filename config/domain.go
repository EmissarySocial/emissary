package config

import "github.com/benpate/steranko"

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	Hostname      string          `json:"hostname"`            // Domain name of a virtual server
	ConnectString string          `json:"connectString"`       // MongoDB connect string
	DatabaseName  string          `json:"databaseName"`        // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	LayoutPath    string          `json:"layoutPath"`          // Path to the directory where the website layout is saved.
	TemplatePaths []string        `json:"templatePaths"`       // Paths to one or more directories where page templates are defined.
	ForwardTo     string          `json:"forwardTo,omitempty"` // Forwarding information for a domain that has moved servers
	Steranko      steranko.Config `json:"steranko,omitempty"`  // Configuration to pass through to Steranko
}
