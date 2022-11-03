package providers

import "github.com/EmissarySocial/emissary/model"

type Null struct{}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (adapter Null) AfterConnect(factory Factory, client *model.Client) error {
	return nil
}

func (adapter Null) AfterUpdate(factory Factory, client *model.Client) error {
	return nil

}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Null) PollStreams(client *model.Client) <-chan model.Stream {
	return nil
}
