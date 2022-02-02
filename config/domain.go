package config

// Domain contains all of the configuration data required to operate a single domain.
type Domain struct {
	Label          string         `path:"label"         json:"label"`               // Human-friendly label for administrators
	Hostname       string         `path:"hostname"      json:"hostname"`            // Domain name of a virtual server
	ConnectString  string         `path:"connectString" json:"connectString"`       // MongoDB connect string
	DatabaseName   string         `path:"databaseName"  json:"databaseName"`        // Name of the MongoDB Database (can be empty string to use default db for the connect string)
	SMTPConnection SMTPConnection `path:"smtp"          json:"smtp"`                // Information for connecting to an SMTP server to send email on behalf of the domain.
	ForwardTo      string         `path:"forwardTo"     json:"forwardTo,omitempty"` // Forwarding information for a domain that has moved servers
	ShowAdmin      bool           `path:"showAdmin"     json:"showAdmin"`           // If TRUE, then show domain settings in admin
	// Steranko       steranko.Config `path:"steranko" json:"steranko"`         // Configuration to pass through to Steranko
}

func NewDomain() Domain {
	return Domain{
		SMTPConnection: SMTPConnection{},
		// Steranko:       steranko.Config{},
	}
}
