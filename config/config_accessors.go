package config

import "github.com/benpate/rosetta/schema"

// Schema returns the data schema for the configuration file.
func Schema() schema.Schema {

	return schema.Schema{
		ID:      "emissary.Server",
		Comment: "Validating schema for a server configuration",
		Element: schema.Object{
			Properties: schema.ElementMap{
				"domains":             schema.Array{Items: DomainSchema()},
				"providers":           schema.Array{Items: ProviderSchema()},
				"templates":           schema.Array{Items: ReadableFolderSchema(), MinLength: 1},
				"emails":              schema.Array{Items: ReadableFolderSchema(), MinLength: 1},
				"attachmentOriginals": WritableFolderSchema(),
				"attachmentCache":     WritableFolderSchema(),
				"certificates":        WritableFolderSchema(),
				"adminEmail":          schema.String{Format: "email"},
				"activityPubCache":    DatabaseConnectInfo(),
			},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (config *Config) GetPointer(name string) (any, bool) {

	switch name {

	case "domains":
		return &config.Domains, true

	case "providers":
		return &config.Providers, true

	case "templates":
		return &config.Templates, true

	case "emails":
		return &config.Emails, true

	case "attachmentOriginals":
		return &config.AttachmentOriginals, true

	case "attachmentCache":
		return &config.AttachmentCache, true

	case "certificates":
		return &config.Certificates, true

	case "adminEmail":
		return &config.AdminEmail, true

	case "activityPubCache":
		return &config.ActivityPubCache, true

	}

	return nil, false
}
