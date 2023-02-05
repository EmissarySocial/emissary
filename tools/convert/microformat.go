package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"willnorris.com/go/microformats"
)

func MicroformatToMessage(feed *microformats.Microformat, entry *microformats.Microformat) model.Message {

	message := model.NewMessage()

	// Get properties from entry
	message.Document.URL = MicroformatPropertyToString(entry, "url")
	message.Document.Label = MicroformatPropertyToString(entry, "name")
	message.Document.Summary = MicroformatPropertyToString(entry, "summary")

	// Get photo from entry, then feed
	if photoURL := MicroformatPropertyToString(entry, "photo"); photoURL != "" {
		message.Document.ImageURL = photoURL
	} else if photoURL := MicroformatPropertyToString(feed, "photo"); photoURL != "" {
		message.Document.ImageURL = photoURL
	}

	// Get author from entry, then feed
	if author := AnyToMicroformat(entry.Properties["author"]); author != nil {
		message.Document.Author = MicroformatToAuthor(author)
	} else if author := AnyToMicroformat(feed.Properties["author"]); author != nil {
		message.Document.Author = MicroformatToAuthor(author)
	}

	// Get the publish date from the entry
	if published := MicroformatPropertyToString(entry, "published"); published != "" {
		if publishDate, err := time.Parse(time.RFC3339, published); err == nil {
			message.Document.PublishDate = publishDate.Unix()
		}
	}

	// Default PublishDate just in case
	if message.Document.PublishDate == 0 {
		message.Document.PublishDate = time.Now().Unix()
	}

	return message
}

func MicroformatToAuthor(entry *microformats.Microformat) model.PersonLink {

	var author model.PersonLink

	author.Name = MicroformatPropertyToString(entry, "name")
	author.ProfileURL = MicroformatPropertyToString(entry, "url")
	author.ImageURL = MicroformatPropertyToString(entry, "photo", "logo")
	author.EmailAddress = MicroformatPropertyToString(entry, "email")

	return author
}

func AnyToMicroformat(value any) *microformats.Microformat {

	switch o := value.(type) {
	case []any:
		if len(o) > 0 {
			return AnyToMicroformat(o[0])
		}

	case *microformats.Microformat:
		return o
	}

	return nil
}

func MicroformatPropertyToString(entry *microformats.Microformat, names ...string) string {

	for _, name := range names {

		if value, ok := entry.Properties[name]; ok {

			for _, item := range value {
				switch o := item.(type) {
				case string:
					return o

				case *microformats.Microformat:
					return o.Value
				}
			}
		}
	}

	return ""
}
