package external

import "github.com/EmissarySocial/emissary/model"

type Null struct{}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Null) PollStreams(client model.Client) error {
	return nil
}

func (adapter Null) PostStream(clent model.Client) error {
	return nil
}
