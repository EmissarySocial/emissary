package render

import (
	"html/template"

	"github.com/benpate/convert"
)

func FuncMap() template.FuncMap {

	return template.FuncMap{
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
	}
}
