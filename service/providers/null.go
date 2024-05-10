package providers

import "github.com/EmissarySocial/emissary/model"

type Null struct{}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (adapter Null) AfterConnect(factory Factory, client *model.Connection) error {
	return nil
}

func (adapter Null) AfterUpdate(factory Factory, client *model.Connection) error {
	return nil

}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Null) PollStreams(client *model.Connection) <-chan model.Stream {
	return nil
}
