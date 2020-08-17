package render

import "reflect"

func FuncMap() map[string]interface{} {

	return map[string]interface{}{
		"date": func(value reflect.Value) string {
			return "date"
		},
	}
}
