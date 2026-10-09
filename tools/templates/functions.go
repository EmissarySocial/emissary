package templates

import (
	"html/template"
	"strings"
	"time"

	"github.com/benpate/color"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/funcmap"
	"github.com/benpate/rosetta/ranges"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dustin/go-humanize"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/EmissarySocial/emissary/tools/groupie"
	"github.com/EmissarySocial/emissary/tools/markdown"
	"github.com/EmissarySocial/emissary/tools/parse"
	"github.com/EmissarySocial/emissary/tools/tinyDate"
	"github.com/benpate/icon"
)

// FuncMap returns every helper function available to Emissary HTML templates
func FuncMap(icons icon.Provider) template.FuncMap {

	result := funcmap.All()

	// RULE: Override rosetta's "markdown" helper so that every Markdown value in
	// every template is converted and sanitized by the application's single
	// converter, instead of rosetta's unsanitized one.
	result["markdown"] = func(value any) template.HTML {
		return template.HTML(markdown.ToHTML(convert.String(value))) // #nosec G203 -- markdown.ToHTML sanitizes its output
	}

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

	// RULE: Escape before wrapping.  This helper returns template.HTML, so html/template
	// will not escape it downstream, and its inputs are plain text -- a search term and the
	// text it was found in -- neither of which is trusted markup.
	result["highlight"] = func(text string, search string) template.HTML {

		escapedText := template.HTMLEscapeString(text)

		if search == "" {
			return template.HTML(escapedText) // #nosec G203 -- escaped immediately above
		}

		escapedSearch := template.HTMLEscapeString(search)
		wrapped := strings.ReplaceAll(escapedText, escapedSearch, `<b class="highlight">`+escapedSearch+"</b>")

		return template.HTML(wrapped) // #nosec G203 -- both halves are escaped above; only the <b> wrapper is markup
	}

	result["collection"] = func(max int, collection streams.Document) (sliceof.Object[streams.Document], error) {

		// Make a channel of the first N documents
		ch := collections.RangeDocuments(collection)
		ch = ranges.Limit(max, ch)

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
		return template.HTML(icons.Get(name)) // #nosec G203 -- Icons.Get returns markup built from a fixed icon set; every call site passes a "token"-validated name
	}

	result["iconFilled"] = func(name string) template.HTML {
		if icons == nil {
			return template.HTML("")
		}
		return template.HTML(icons.Get(name + "-fill")) // #nosec G203 -- Icons.Get returns markup built from a fixed icon set; every call site passes a "token"-validated name
	}

	return result
}
