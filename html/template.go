package template

import "html/template"

type HTML struct {
	Label    string
	Category string
	Template template.Template
}
