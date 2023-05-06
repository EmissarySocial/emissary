package convert

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
)

// ActivityPubPersonLink converts a streams.Document into a model.PersonLink
func ActivityPubPersonLink(person streams.Document) model.PersonLink {

	person, _ = person.AsObject()

	return model.PersonLink{
		Name:       person.Name(),
		ProfileURL: person.ID(),
		ImageURL:   person.ImageURL(),
	}
}
