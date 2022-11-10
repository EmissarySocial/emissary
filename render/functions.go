package render

import (
	"encoding/json"
	"html/template"
	"time"

	"github.com/benpate/icon"
	"github.com/benpate/rosetta/convert"
	humanize "github.com/dustin/go-humanize"
)

func FuncMap(icons icon.Provider) template.FuncMap {

	return template.FuncMap{
		"icon": func(name string) template.HTML {
			return template.HTML(icons.Get(name))
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

		"head": func(slice List) Renderer { // Returns the first item in a resultSet
			return slice[0]
		},

		"last": func(slice List) Renderer { // Returns the last item in a resultSet
			return slice[len(slice)-1]
		},

		"tail": func(slice List) List { // Returns all but the first item in a resultSet
			length := len(slice)
			if length == 0 {
				return List{}
			}
			return slice[1:]
		},

		"removeLast": func(slice List) List { // Returns all but the last item in a resultSet
			length := len(slice)
			if length == 0 {
				return List{}
			}
			return slice[:length-1]
		},

		"reverse": func(slice List) List { // Returns a new resultSet with reverse ordering
			length := len(slice)
			result := make(List, length)
			for index := range slice {
				result[length-1-index] = slice[index]
			}
			return result
		},

		"isEmpty": func(slice List) bool { // Returns true if there are NO records in the resultset
			return len(slice) == 0
		},

		"isSingle": func(slice List) bool { // Returns true if there are NO records in the resultset
			return len(slice) == 1
		},

		"notEmpty": func(slice List) bool { // Returns true if there are records in the resultset
			return len(slice) > 0
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
			return humanize.Time(time.UnixMilli(valueInt))
		},
	}
}
