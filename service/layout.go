package service

import (
	"html/template"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/list"
)

type Layout struct {
	path     string
	Template *template.Template
}

func NewLayout(path string) (Layout, error) {

	layout := Layout{
		path:     path,
		Template: template.New(""),
	}

	files, err := ioutil.ReadDir(path)

	if err != nil {
		return layout, derp.Wrap(err, "ghost.service.NewLayout", "Unable to list files in filesystem")
	}

	for _, file := range files {

		contents, err := ioutil.ReadFile(path + "/" + file.Name())

		if err != nil {
			return layout, derp.Wrap(err, "ghost.service.NewLayout", "Error reading file from filesystem", file.Name())
		}

		templateName := list.Head(file.Name(), ".")
		t, err := template.New("").Parse(string(contents))

		if err != nil {
			return layout, derp.Wrap(err, "ghost.service.NewLayout", "Error parsing template file", file.Name(), string(contents))
		}

		// Try to append this layout to the ParseTree
		if _, err := layout.Template.AddParseTree(templateName, t.Tree); err != nil {
			return layout, derp.Wrap(err, "ghost.service.NewLayout", "Error adding parseTree")
		}
	}

	return layout, nil
}
