package render

import "reflect"

// FuncMap returns a library of functions that can be used in Templates.
func FuncMap() map[string]interface{} {

	return map[string]interface{}{
		"date": func(value reflect.Value) string {
			return "date"
		},
	}
}
