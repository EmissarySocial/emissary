package service

type GeocodeTimezone struct {
	connectionService *Connection
	hostname          string
}

func NewGeocodeTimezone(connectionService *Connection, hostname string) GeocodeTimezone {
	return GeocodeTimezone{
		connectionService: connectionService,
		hostname:          hostname,
	}
}
