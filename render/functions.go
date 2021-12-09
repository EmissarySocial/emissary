package render

import "html/template"

func FuncMap() template.FuncMap {

	return template.FuncMap{
		"first": func(slice []Stream) Stream {
			return slice[0]
		},
		"last": func(slice []Stream) Stream {
			return slice[len(slice)-1]
		},
	}
}
