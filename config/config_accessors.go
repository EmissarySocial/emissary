package config

func (config *Config) GetObjectOK(name string) (any, bool) {

	switch name {

	case "domains":
		return &config.Domains, true

	case "providers":
		return &config.Providers, true

	case "templates":
		return &config.Templates, true

	case "layouts":
		return &config.Layouts, true

	case "emails":
		return &config.Emails, true

	case "attachmentOriginals":
		return &config.AttachmentOriginals, true

	case "attachmentCache":
		return &config.AttachmentCache, true

	case "certificates":
		return &config.Certificates, true

	}

	return nil, false
}

func (config Config) GetStringOK(name string) (string, bool) {

	switch name {

	case "adminEmail":
		return config.AdminEmail, true

	}

	return "", false
}

func (config *Config) SetStringOK(name string, value string) bool {

	switch name {

	case "adminEmail":
		config.AdminEmail = value
		return true

	}

	return false
}
