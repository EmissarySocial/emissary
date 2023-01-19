package config

func (owner Owner) GetStringOK(name string) (string, bool) {

	switch name {

	case "displayName":
		return owner.DisplayName, true

	case "username":
		return owner.Username, true

	case "emailAddress":
		return owner.EmailAddress, true

	case "phoneNumber":
		return owner.PhoneNumber, true

	case "mailingAddress":
		return owner.MailingAddress, true
	}

	return "", false
}

func (owner *Owner) SetStringOK(name string, value string) bool {

	switch name {

	case "displayName":
		owner.DisplayName = value
		return true

	case "username":
		owner.Username = value
		return true

	case "emailAddress":
		owner.EmailAddress = value
		return true

	case "phoneNumber":
		owner.PhoneNumber = value
		return true

	case "mailingAddress":
		owner.MailingAddress = value
		return true
	}

	return false
}
