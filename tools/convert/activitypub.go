package convert

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/davecgh/go-spew/spew"
)

func ActivityPubPersonLinks(person streams.Document) []model.PersonLink {

	spew.Dump("ActivityPubPersonLinks -------------", person.Value())

	result := make([]model.PersonLink, 0)

	for ; !person.IsNil(); person = person.Next() {
		if details, err := person.Load(); err == nil {
			link := model.PersonLink{
				Name:       details.Name(),
				ProfileURL: details.ID(),
				ImageURL:   details.ImageURL(),
			}
			result = append(result, link)
		} else {
			derp.Report(err)
		}
	}

	spew.Dump(result)

	return result
}

// ActivityPubPersonLink converts a streams.Document into a model.PersonLink
func ActivityPubPersonLink(person streams.Document) model.PersonLink {

	person, err := person.Load()

	derp.Report(err)

	return model.PersonLink{
		Name:       person.Name(),
		ProfileURL: person.ID(),
		ImageURL:   person.ImageURL(),
	}
}
