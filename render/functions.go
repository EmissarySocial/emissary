package render

import "html/template"

func FuncMap() template.FuncMap {

	return template.FuncMap{
		"head": func(slice []Stream) Stream { // Returns the first item in a resultSet
			return slice[0]
		},
		"last": func(slice []Stream) Stream { // Returns the last item in a resultSet
			return slice[len(slice)-1]
		},
		"tail": func(slice []Stream) []Stream { // Returns all but the first item in a resultSet
			length := len(slice)
			if length == 0 {
				return []Stream{}
			}
			return slice[1:]
		},
		"removeLast": func(slice []Stream) []Stream { // Returns all but the last item in a resultSet
			length := len(slice)
			if length == 0 {
				return []Stream{}
			}
			return slice[:length-1]
		},
		"reverse": func(slice []Stream) []Stream { // Returns a new resultSet with reverse ordering
			length := len(slice)
			result := make([]Stream, length)
			for index := range slice {
				result[length-1-index] = slice[index]
			}
			return result
		},
		"notEmpty": func(slice []Stream) bool { // Returns true if there are records in the resultset
			return len(slice) > 0
		},
	}
}
