package render

import "html/template"

func FuncMap() template.FuncMap {

	return template.FuncMap{
		"first": func(slice []Renderer) Renderer {
			return slice[0]
		},
		"last": func(slice []Renderer) Renderer {
			return slice[len(slice)-1]
		},
	}
}
