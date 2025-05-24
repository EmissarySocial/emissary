package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

type Null struct{}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (adapter Null) AfterConnect(factory Factory, client *model.Connection, vault mapof.String) error {
	return nil
}

func (adapter Null) AfterUpdate(factory Factory, client *model.Connection, vault mapof.String) error {
	return nil

}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Null) PollStreams(client *model.Connection) <-chan model.Stream {
	return nil
}
