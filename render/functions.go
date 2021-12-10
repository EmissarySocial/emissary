package render

import "html/template"

func FuncMap() template.FuncMap {

	return template.FuncMap{
		"first": func(slice []Stream) Stream { // Returns the first item in a resultSet
			return slice[0]
		},
		"last": func(slice []Stream) Stream { // Returns the last item in a resultSet
			return slice[len(slice)-1]
		},
		"reverse": func(slice []Stream) []Stream { // Returns a new resultSet with reverse ordering
			length := len(slice)
			result := make([]Stream, length)
			for index := range slice {
				result[length-1-index] = slice[index]
			}
			return result
		},
	}
}
