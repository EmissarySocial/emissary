package convert

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// ActivityPubPersonLink converts a streams.Document into a model.PersonLink
func ActivityPubPersonLink(person streams.Document) model.PersonLink {

	person, err := person.Load()

	derp.Report(err)

	return model.PersonLink{
		Name:       person.Name(),
		ProfileURL: person.ID(),
		ImageURL:   person.IconOrImage().URL(),
	}
}
