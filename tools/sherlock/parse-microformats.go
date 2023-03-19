package sherlock

import (
	"io"
	"net/url"
	"time"

	"github.com/benpate/rosetta/convert"
	"willnorris.com/go/microformats"
)

func ParseMicroFormats(uri *url.URL, reader io.Reader, data *Page) {

	mf := microformats.Parse(reader, uri)

	for _, item := range mf.Items {
		for _, property := range item.Type {
			switch property {

			// https://microformats.org/wiki/h-entry
			case "h-entry":

				if data.CanonicalURL == "" {
					data.CanonicalURL = convert.String(item.Properties["url"])
				}

				if data.CanonicalURL == "" {
					data.CanonicalURL = convert.String(item.Properties["uid"])
				}

				if data.Title == "" {
					data.Title = convert.String(item.Properties["name"])
				}

				if data.Description == "" {
					data.Description = convert.String(item.Properties["summary"])
				}

				if data.Image.URL == "" {
					data.Image.URL = convert.String(item.Properties["photo"])
				}

				if data.PublishDate == 0 {
					if publishedString := convert.String(item.Properties["published"]); publishedString != "" {
						if published, err := time.Parse(time.RFC3339, publishedString); err == nil {
							data.PublishDate = published.Unix()
						}
					}
				}

				if tags := convert.SliceOfString(item.Properties["category"]); len(tags) > 0 {
					data.Tags = append(data.Tags, tags...)
				}

				if data.InReplyTo == "" {
					data.InReplyTo = convert.String(item.Properties["in-reply-to"])
				}

				if data.InReplyTo == "" {
					data.InReplyTo = convert.String(item.Properties["like-of"])
				}

				if data.InReplyTo == "" {
					data.InReplyTo = convert.String(item.Properties["repost-of"])
				}

				// Look through Child Items for Author information https://microformats.org/wiki/h-card
				for _, child := range item.Children {
					for _, childProperty := range child.Type {
						switch childProperty {
						case "h-card":
							data.Authors = append(data.Authors, Author{
								Name:  convert.String(child.Properties["name"]),
								URL:   convert.String(child.Properties["url"]),
								Email: convert.String(child.Properties["email"]),
								Image: Image{
									URL: convert.String(child.Properties["photo"]),
								},
							})
						}
					}
				}
			}
		}
	}
}

/* OLD CODE TO DOUBLE CHECK.  DID I MISS ANYTHING??


func populateMention(mf *microformats.Data, mention *model.Mention) {

	spew.Dump("populateMentions", mf, mention)

	for _, item := range mf.Items {

		for _, itemType := range item.Type {

			switch itemType {

			// Parse author information [https://microformats.org/wiki/h-card]
			case "h-card":

				if mention.Author.Name == "" {
					mention.Author.Name = convert.String(item.Properties["name"])
				}

				if mention.Author.Name == "" {
					mention.Author.Name = convert.String(item.Properties["given-name"])
				}

				if mention.Author.Name == "" {
					mention.Author.Name = convert.String(item.Properties["nickname"])
				}

				if mention.Author.ProfileURL == "" {
					mention.Author.ProfileURL = convert.String(item.Properties["url"])
				}

				if mention.Author.EmailAddress == "" {
					mention.Author.EmailAddress = convert.String(item.Properties["email"])
				}

				if mention.Author.ImageURL == "" {
					mention.Author.ImageURL = convert.String(item.Properties["photo"])
				}

				if mention.Author.ImageURL == "" {
					mention.Author.ImageURL = convert.String(item.Properties["logo"])
				}

				continue

			// Parse entry data
			case "h-entry": // [https://microformats.org/wiki/h-entry]

				if mention.Origin.Label == "" {
					mention.Origin.Label = convert.String(item.Properties["name"])
				}

				if mention.Origin.Summary == "" {
					mention.Origin.Summary = convert.String(item.Properties["summary"])
				}

				if mention.Origin.ImageURL == "" {
					mention.Origin.ImageURL = convert.String(item.Properties["photo"])
				}
			}
		}
	}

	// Last, scan global values for data that may not have been found in the h-entry
	if mention.Author.ProfileURL == "" {
		if me, ok := mf.Rels["me"]; ok {
			mention.Author.ProfileURL = convert.String(me)
		}
	}
}


*/
