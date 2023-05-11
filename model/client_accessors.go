package model

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (client Client) GetPointer(name string) (any, bool) {
	switch name {

	case "active":
		return &client.Active, true

	case "data":
		return &client.Data, true

	case "providerId":
		return &client.ProviderID, true
	}

	return nil, false
}
