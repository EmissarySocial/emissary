package convert

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
)

// ActivityPubAttributedTo generates a model.PersonLink for the first valid record in AttributedTo
func ActivityPubAttributedTo(document streams.Document) model.PersonLink {

	for attributedTo := document.AttributedTo(); attributedTo.NotNil(); attributedTo = attributedTo.Tail() {
		if person, err := attributedTo.Load(); err == nil {
			return model.PersonLink{
				Name:       person.Name(),
				ProfileURL: person.ID(),
				IconURL:    person.IconOrImage().URL(),
			}
		}
	}

	return model.NewPersonLink()
}
