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
