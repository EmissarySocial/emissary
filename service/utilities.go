package service

import (
	"encoding/json"
	"html/template"
	"io/fs"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/list"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// notDeleted ensures that a criteria expression does not include soft-deleted items.
func notDeleted(criteria exp.Expression) exp.Expression {
	return criteria.And(exp.Equal("journal.deleteDate", 0))
}

// loadHTMLTemplateFromFilesystem locates and parses a Template sub-directory within the filesystem path
func loadHTMLTemplateFromFilesystem(filesystem fs.FS, t *template.Template, funcMap template.FuncMap) error {

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// List all files in the target directory
	files, err := fs.ReadDir(filesystem, ".")

	if err != nil {
		return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to list directory", filesystem)
	}

	for _, file := range files {

		filename := file.Name()
		actionBytes, extension := list.Dot(filename).SplitTail()
		actionID := actionBytes.String()

		// Only HTML files beyond this point...
		if extension == "html" {

			// Try to read the file from the filesystem
			content, err := fs.ReadFile(filesystem, filename)

			if err != nil {
				return derp.Report(derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Cannot read file", filename))
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Funcs(funcMap).Parse(contentString)

			if err != nil {
				return derp.Report(derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to parse template HTML", contentString))
			}

			// Add this minified template into the resulting parse-tree
			t.AddParseTree(actionID, contentTemplate.Tree)
		}
	}

	// Return to caller.
	return nil
}

// loadModelFromFilesystem locates and parses a schema from the filesystem path
func loadModelFromFilesystem(filesystem fs.FS, model any, location string) error {

	file, err := fs.ReadFile(filesystem, "schema.json")

	if err != nil {
		return derp.Wrap(err, "service.loadModelFromFilesystem", "Cannot read file: schema.json", location)
	}

	// Unmarshal the file into the schema.
	if err := json.Unmarshal(file, model); err != nil {
		return derp.Wrap(err, "service.loadModelFromFilesystem", "Invalid JSON configuration file: schema.json", location)
	}

	// Return to caller.
	return nil
}

func value[T any](value T, _ bool) T {
	return value
}
