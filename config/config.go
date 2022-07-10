package config

// Config defines all of the domains available on this server
type Config struct {
	AdminURL            string     `path:"adminUrl"            json:"adminUrl"`            // Path to use for the server admin console (if blank, then console is not available)
	AdminEmail          string     `path:"adminEmail"          json:"adminEmail"`          // Email address of the server owner - also used as primary contact for this server
	AdminPassword       string     `path:"adminPassword"       json:"adminPassword"`       // Password for access to domain admin console
	Domains             DomainList `path:"domains"             json:"domains"`             // Slice of one or more domain configurations
	Certificates        Folder     `path:"certificates"        json:"certificates"`        // Folder containing the SSL certificate cache for Let's Encrypt AutoSSL
	Templates           Folder     `path:"templates"           json:"templates"`           // Folder containing all stream templates
	Layouts             Folder     `path:"layouts"             json:"layouts"`             // Folder containing all system layouts
	Static              Folder     `path:"static"              json:"static"`              // Folder containing all attachments
	AttachmentOriginals Folder     `path:"attachmentOriginals" json:"attachmentOriginals"` // Folder where original attachments will be stored
	AttachmentCache     Folder     `path:"attachmentCache"     json:"attachmentCache"`     // Folder (possibly memory cache) where cached versions of attachmented files will be stored.
}

// NewConfig returns a fully initialized (but empty) Config data structure.
func NewConfig() Config {

	return Config{
		Domains: make(DomainList, 0),
	}
}

/*
func NewDefaultConfig() Config {

	fmt.Println("Generating new default configuration...")

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
	config := DefaultConfig(defaultAdminURL, defaultPassword)
	configString, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		panic("Error marshaling new config file: " + err.Error())
	}

	// Try to write the new configuration to the file system
	if err := ioutil.WriteFile(CONFIG_FILENAME, configString, 0777); err != nil {
		panic("Error writing default configuration: " + err.Error() + "\n Check file permissions.")
	}

	// Output helpful hints for system admins
	fmt.Println("Default config file created at: " + CONFIG_FILENAME)
	fmt.Println("")
	fmt.Println("SAVE THIS INFORMATION SOMEWHERE SAFE")
	fmt.Println("-----------------------------------------")
	fmt.Println("To access the server admin, and configure")
	fmt.Println("websites on this server open this URL:")
	fmt.Println("http://localhost/" + config.AdminURL)
	fmt.Println("The admin password is: " + config.AdminPassword)
	fmt.Println("-----------------------------------------")

	// On first run, open web browser in admin mode
	browser.OpenURL("http://localhost/" + config.AdminURL + "?first=true")

	// Success!
	result <- config
}
*/

// DefaultConfig return sthe default configuration for this application.
func DefaultConfig(adminURL string, adminPassword string) Config {

	// If Admin URL is empty, then blank out the password, too
	if adminURL == "" {
		adminPassword = ""
	} else {
		// Otherwise, add a prefix to be clear that there's no overlap with Stream URLs
		adminURL = "." + adminURL
	}

	return Config{
		AdminURL:      adminURL,
		AdminPassword: adminPassword,
		Domains: DomainList{{
			Label:     "Administration Console",
			Hostname:  "localhost",
			ShowAdmin: true,
		}},
		Layouts: Folder{
			Adapter:  "FILE",
			Location: "./_layouts/",
			Sync:     true,
		},
		Static: Folder{
			Adapter:  "FILE",
			Location: "./_static/",
		},
		Templates: Folder{
			Adapter:  "FILE",
			Location: "./_templates/",
			Sync:     true,
		},
		Certificates: Folder{
			Adapter:  "FILE",
			Location: "./_certificates/",
			Sync:     false,
		},
		AttachmentOriginals: Folder{
			Adapter:  "FILE",
			Location: "./_attachments/originals",
			Sync:     false,
		},
		AttachmentCache: Folder{
			Adapter:  "FILE",
			Location: "./_attachments/cache",
			Sync:     false,
		},
	}
}

/************************
 * Data Accessors
 ************************/

// DomainNames returns an array of domains names in this configuration.
func (config Config) DomainNames() []string {

	result := make([]string, len(config.Domains))

	for index := range config.Domains {
		result[index] = config.Domains[index].Hostname
	}

	return result
}
