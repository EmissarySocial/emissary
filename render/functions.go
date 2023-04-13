package render

import (
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/tinyDate"
	"github.com/benpate/icon"
	"github.com/benpate/rosetta/convert"
	humanize "github.com/dustin/go-humanize"
)

func FuncMap(icons icon.Provider) template.FuncMap {

	return template.FuncMap{

		"add": func(a any, b any) int {
			return convert.Int(a) + convert.Int(b)
		},

		"subtract": func(a any, b any) int {
			return convert.Int(a) - convert.Int(b)
		},

		"divide": func(a any, b any) int64 {
			return convert.Int64(a) / convert.Int64(b)
		},

		"icon": func(name string) template.HTML {
			return template.HTML(icons.Get(name))
		},

		"iconFilled": func(name string) template.HTML {
			return template.HTML(icons.Get(name + "-fill"))
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

		"html": func(value string) template.HTML {
			return template.HTML(value)
		},

		"css": func(value string) template.CSS {
			return template.CSS(value)
		},

		"json": func(value any) string {
			result, _ := json.MarshalIndent(value, "", "  ")
			return string(result)
		},

		"isoDate": func(value any) string {

			valueInt := convert.Int64(value)

			if valueInt == 0 {
				return ""
			}

			return time.Unix(valueInt, 0).Format(time.RFC3339)
		},

		"humanizeTime": func(value any) string {
			valueInt := convert.Int64(value)
			return humanize.Time(time.Unix(valueInt, 0))
		},

		"tinyDate": func(value any) string {
			valueInt := convert.Int64(value)
			if valueInt == 0 {
				return ""
			}
			return tinyDate.FormatDiff(time.Unix(valueInt, 0), time.Now())
		},

		"longDate": func(value any) string {
			valueInt := convert.Int64(value)
			if valueInt == 0 {
				return ""
			}
			return time.Unix(valueInt, 0).Format("Monday, January 2, 2006")
		},
	}
}
