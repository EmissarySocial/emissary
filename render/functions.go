package render

import (
	"encoding/json"
	"html/template"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/html"
	"github.com/davecgh/go-spew/spew"

	"github.com/EmissarySocial/emissary/tools/tinyDate"
	"github.com/benpate/icon"
	"github.com/benpate/rosetta/convert"
	humanize "github.com/dustin/go-humanize"
)

func FuncMap(icons icon.Provider) template.FuncMap {

	return template.FuncMap{

		"seq": func(count int) []int {
			result := make([]int, count)
			for i := 0; i < count; i++ {
				result[i] = i
			}
			return result
		},

		"max": func(values ...any) int {
			var result int = math.MinInt
			for _, value := range values {
				if value32 := convert.Int(value); value32 > result {
					result = value32
				}
			}
			return result
		},

		"min": func(values ...any) int {
			var result int = math.MaxInt
			for _, value := range values {
				if value32 := convert.Int(value); value32 < result {
					result = value32
				}
			}
			return result
		},

		"array": func(values ...any) []any {
			return values
		},

		"add": func(a any, b any) int {
			return convert.Int(a) + convert.Int(b)
		},

		"subtract": func(a any, b any) int {
			return convert.Int(a) - convert.Int(b)
		},

		"divide": func(a any, b any) int64 {
			return convert.Int64(a) / convert.Int64(b)
		},

		"first": func(values ...any) any {
			for _, value := range values {
				if compare.NotZero(value) {
					return value
				}
			}
			return nil
		},

		"icon": func(name string) template.HTML {
			return template.HTML(icons.Get(name))
		},

		"iconFilled": func(name string) template.HTML {
			return template.HTML(icons.Get(name + "-fill"))
		},

		"iif": func(condition bool, trueValue any, falseValue any) any {
			if condition {
				return trueValue
			}
			return falseValue
		},

		"pluralize": func(count any, single string, plural string) string {
			countInt := convert.Int(count)
			if countInt == 1 {
				return single
			}
			return plural
		},

		"lowerCase": func(name any) string {
			return strings.ToLower(convert.String(name))
		},

		"dollarFormat": func(value any) string {

			var unitAmount int64

			switch value := value.(type) {
			case float32:
				unitAmount = int64(value * 100)
			case float64:
				unitAmount = int64(value * 100)
			default:
				unitAmount = convert.Int64(value)
			}

			stringValue := convert.String(unitAmount)
			length := len(stringValue)
			for length < 3 {
				stringValue = "0" + stringValue
				length = len(stringValue)
			}
			return "$" + stringValue[:length-2] + "." + stringValue[length-2:]
		},

		"removeLinks": func(value string) template.HTML {
			result := strings.ReplaceAll(value, "<a ", "<span ")
			result = strings.ReplaceAll(result, "</a", "</span")
			return template.HTML(result)
		},

		"textOnly": html.RemoveTags,

		"summary": html.Summary,

		"html": func(value string) template.HTML {
			return template.HTML(value)
		},

		"htmlMinimal": func(value string) template.HTML {
			return template.HTML(html.Minimal(value))
		},

		"attr": func(value string) template.HTMLAttr {
			return template.HTMLAttr(value)
		},

		"css": func(value string) template.CSS {
			return template.CSS(value)
		},

		"json": func(value any) string {
			result, _ := json.MarshalIndent(value, "", "    ")
			return string(result)
		},

		"now": time.Now,

		"isoDate": func(value any) string {

			if valueTime, ok := convert.TimeOk(value, time.Time{}); ok {
				return valueTime.Format(time.RFC3339)
			}

			return ""
		},

		"epochDate": func(value any) int64 {
			return convert.Time(value).Unix()
		},

		"humanizeTime": func(value any) string {
			valueTime := convert.Time(value)
			return humanize.Time(valueTime)
		},

		"tinyDate": func(value any) string {
			valueTime := convert.Time(value)
			if valueTime.IsZero() {
				return ""
			}
			return tinyDate.FormatDiff(valueTime, time.Now())
		},

		"hasImage": func(value string) bool {
			if strings.Contains(value, "<img") {
				return true
			}

			if strings.Contains(value, "<picture") {
				return true
			}

			return false
		},

		"shortDate": func(value any) string {
			valueTime := convert.Time(value)
			if valueTime.IsZero() {
				return ""
			}
			return valueTime.Format("Jan 2, 2006")
		},

		"longDate": func(value any) string {
			valueTime := convert.Time(value)
			if valueTime.IsZero() {
				return ""
			}
			return valueTime.Format("Monday, January 2, 2006")
		},

		"shortTime": func(value any) string {
			valueTime := convert.Time(value)

			if valueTime.IsZero() {
				return ""
			}
			return valueTime.Format("3:04:05 PM")
		},

		"addQueryParams": func(extraParams string, url string) string {
			if strings.Contains(url, "?") {
				return url + "&" + extraParams
			}
			return url + "?" + extraParams
		},

		"queryEscape": func(value string) string {
			return url.QueryEscape(value)
		},

		"dump": func(values ...any) string {
			for _, value := range values {
				spew.Dump(value)
			}
			return ""
		},

		"collection": func(max int, collection streams.Document) ([]streams.Document, error) {

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
		},
	}
}
