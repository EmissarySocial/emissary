package templates

import (
	"html/template"
	"strings"
	"time"

	"github.com/benpate/color"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/funcmap"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dustin/go-humanize"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/EmissarySocial/emissary/tools/groupie"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/tinyDate"
	"github.com/benpate/icon"
)

func FuncMap(icons icon.Provider) template.FuncMap {

	result := funcmap.All()

	result["humanizeTime"] = func(value any) string {
		valueTime := convert.Time(value)
		return humanize.Time(valueTime)
	}

	result["tinyDate"] = func(value any) string {
		valueTime := convert.Time(value)
		if valueTime.IsZero() {
			return ""
		}
		return tinyDate.FormatDiff(valueTime, time.Now())
	}

	result["parseColor"] = func(value string) color.Color {
		return color.Parse(value)
	}

	result["highlight"] = func(text string, search string) template.HTML {
		if search == "" {
			return template.HTML(text)
		}
		result := strings.ReplaceAll(text, search, `<b class="highlight">`+search+"</b>")
		return template.HTML(result)
	}

	result["collection"] = func(max int, collection streams.Document) (sliceof.Object[streams.Document], error) {

		// Make a channel of the first N documents
		done := make(chan struct{})
		ch := collections.Documents(collection, done)
		ch = channel.Limit(max, ch, done)

		// Read all of the documents from the channel
		result := make([]streams.Document, 0, max)
		for document := range ch {
			result = append(result, document.UnwrapActivity())
		}

		// Return the result.
		return result, nil
	}

	result["parseTags"] = func(value string) sliceof.String {
		return parse.Hashtags(value)
	}

	result["groupie"] = func() *groupie.Groupie {
		return groupie.New()
	}

	result["replaceTags"] = func(value string, tags []any) string {

		for _, tag := range tags {
			if replacer, isReplacer := tag.(Replacer); isReplacer {
				value = replacer.Replace(value)
			}
		}

		return value
	}

	result["nilObjectID"] = func() primitive.ObjectID {
		return primitive.NilObjectID
	}

	result["newObjectID"] = func() string {
		return primitive.NewObjectID().Hex()
	}

	result["icon"] = func(name string) template.HTML {
		if icons == nil {
			return template.HTML("")
		}
		return template.HTML(icons.Get(name))
	}

	result["iconFilled"] = func(name string) template.HTML {
		return template.HTML(icons.Get(name + "-fill"))
	}

	return result
}
