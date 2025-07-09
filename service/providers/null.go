package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

type Null struct{}

/******************************************
 * Lifecycle Methods
 ******************************************/

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

func (adapter Null) PollStreams(connection *model.Connection) <-chan model.Stream {
	return nil
}
