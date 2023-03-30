package model

func (client Client) GetBoolOK(name string) (bool, bool) {
	switch name {

	case "active":
		return client.Active, true
	}

	return false, false
}

func (client Client) GetObject(name string) (any, bool) {
	switch name {

	case "data":
		return &client.Data, true
	}

	return nil, false
}

func (client Client) GetString(name string) string {
	result, _ := client.GetStringOK(name)
	return result
}

func (client Client) GetStringOK(name string) (string, bool) {
	switch name {

	case "providerId":
		return client.ProviderID, true
	}

	return "", false
}

func (client *Client) SetBool(name string, value bool) bool {
	switch name {

	case "active":
		client.Active = value
		return true
	}

	return false
}

func (client *Client) SetString(name string, value string) bool {
	switch name {

	case "providerId":
		client.ProviderID = value
		return true
	}

	return false
}
