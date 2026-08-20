package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

// Null is a do-nothing Provider, used for Connections that need no external service
type Null struct{}

/******************************************
 * Lifecycle Methods
 ******************************************/

// BeforeSave applies any last-minute changes to this Connection before it is written to the database
func (adapter Null) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect applies any extra changes required when this Connection is first activated
func (adapter Null) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Null) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Null) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}

/******************************************
 * Adapter Methods
 ******************************************/

// PollStreams implements the polling Adapter interface. The Null provider never produces any Streams.
func (adapter Null) PollStreams(connection *model.Connection) <-chan model.Stream {
	return nil
}
