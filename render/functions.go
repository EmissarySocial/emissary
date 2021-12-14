package render

import "html/template"

func FuncMap() template.FuncMap {

	return template.FuncMap{
		"head": func(slice []Renderer) Renderer { // Returns the first item in a resultSet
			return slice[0]
		},
		"last": func(slice []Renderer) Renderer { // Returns the last item in a resultSet
			return slice[len(slice)-1]
		},
		"tail": func(slice []Renderer) []Renderer { // Returns all but the first item in a resultSet
			length := len(slice)
			if length == 0 {
				return []Renderer{}
			}
			return slice[1:]
		},
		"removeLast": func(slice []Renderer) []Renderer { // Returns all but the last item in a resultSet
			length := len(slice)
			if length == 0 {
				return []Renderer{}
			}
			return slice[:length-1]
		},
		"reverse": func(slice []Renderer) []Renderer { // Returns a new resultSet with reverse ordering
			length := len(slice)
			result := make([]Renderer, length)
			for index := range slice {
				result[length-1-index] = slice[index]
			}
			return result
		},
		"notEmpty": func(slice []Renderer) bool { // Returns true if there are records in the resultset
			return len(slice) > 0
		},
	}
}
