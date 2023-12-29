package step

import (
	"text/template"
	"time"

	"github.com/benpate/rosetta/convert"
)

func FuncMap() template.FuncMap {

	return map[string]any{
		"now": time.Now,
		"shortTime": func(value any) string {
			valueTime := convert.Time(value)
			return valueTime.Format("3:04:05 PM")
		},
	}
}
