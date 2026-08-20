package config

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

// Schema returns the data schema for the configuration file.
func Schema() schema.Schema {

	return schema.Schema{
		ID:      "emissary.Server",
		Comment: "Validating schema for a server configuration",
		Element: schema.Object{
			Properties: schema.ElementMap{
				"domains":              schema.Array{Items: DomainSchema()},
				"templates":            schema.Array{Items: ReadableFolderSchema("templates"), MinLength: 1},
				"attachmentOriginals":  WritableFolderSchema("attachmentOriginals"),
				"attachmentCache":      WritableFolderSchema("attachmentCache"),
				"exportCache":          WritableFolderSchema("exportCache"),
				"certificates":         WritableFolderSchema("certificates"),
				"debugLevel":           schema.String{Enum: []string{"None", "Trace", "Debug", "Info", "Warn", "Error"}, Default: "Info"},
				"adminEmail":           schema.String{Format: "email"},
				"httpPort":             schema.Integer{Maximum: null.NewInt64(65535), Default: null.NewInt64(80)},
				"httpsPort":            schema.Integer{Maximum: null.NewInt64(65535), Default: null.NewInt64(443)},
				"activityPubCache":     DatabaseConnectInfo(),
				"clientIPStrategy":     schema.String{Enum: []string{"SINGLE-IP-HEADER", "RIGHTMOST-TRUSTED-COUNT", "REMOTE-ADDR"}, Default: "REMOTE-ADDR"},
				"clientIPTrustedCount": schema.Integer{Minimum: null.NewInt64(0), Default: null.NewInt64(0)},
				"clientIPHeader":       schema.String{Default: "X-Real-IP"},
				"trustForwardedHost":   schema.Boolean{},
				"allowPrivateIPs":      schema.Boolean{},
			},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (config *Config) GetPointer(name string) (any, bool) {

	switch name {

	case "domains":
		return &config.Domains, true

	case "templates":
		return &config.Templates, true

	case "attachmentOriginals":
		return &config.AttachmentOriginals, true

	case "attachmentCache":
		return &config.AttachmentCache, true

	case "exportCache":
		return &config.ExportCache, true

	case "certificates":
		return &config.Certificates, true

	case "debugLevel":
		return &config.DebugLevel, true

	case "adminEmail":
		return &config.AdminEmail, true

	case "httpPort":
		return &config.HTTPPort, true

	case "httpsPort":
		return &config.HTTPSPort, true

	case "activityPubCache":
		return &config.ActivityPubCache, true

	case "clientIPStrategy":
		return &config.ClientIPStrategy, true

	case "clientIPTrustedCount":
		return &config.ClientIPTrustedCount, true

	case "clientIPHeader":
		return &config.ClientIPHeader, true

	case "trustForwardedHost":
		return &config.TrustForwardedHost, true

	case "allowPrivateIPs":
		return &config.AllowPrivateIPs, true
	}

	return nil, false
}
