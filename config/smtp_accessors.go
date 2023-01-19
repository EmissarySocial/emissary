package config

func (smtp SMTPConnection) GetBoolOK(name string) (bool, bool) {

	switch name {

	case "tls":
		return smtp.TLS, true
	}

	return false, false
}

func (smtp SMTPConnection) GetIntOK(name string) (int, bool) {

	switch name {

	case "port":
		return smtp.Port, true
	}

	return 0, false
}

func (smtp SMTPConnection) GetStringOK(name string) (string, bool) {

	switch name {

	case "hostname":
		return smtp.Hostname, true

	case "username":
		return smtp.Username, true

	case "password":
		return smtp.Password, true
	}

	return "", false
}

func (smtp *SMTPConnection) SetBoolOK(name string, value bool) bool {

	switch name {

	case "tls":
		smtp.TLS = value
		return true
	}

	return false
}

func (smtp *SMTPConnection) SetIntOK(name string, value int) bool {

	switch name {

	case "port":
		smtp.Port = value
		return true

	}

	return false
}

func (smtp *SMTPConnection) SetStringOK(name string, value string) bool {

	switch name {

	case "hostname":
		smtp.Hostname = value
		return true

	case "username":
		smtp.Username = value
		return true

	case "password":
		smtp.Password = value
		return true

	}

	return false
}
