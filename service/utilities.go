package service

import (
	"encoding/json"
	"html/template"
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/list"
	"github.com/spf13/afero"
	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// notDeleted ensures that a criteria expression does not include soft-deleted items.
func notDeleted(criteria exp.Expression) exp.Expression {
	return criteria.And(exp.Equal("journal.deleteDate", 0))
}

// loadHTMLTemplateFromFilesystem locates and parses a Template sub-directory within the filesystem path
func loadHTMLTemplateFromFilesystem(filesystem afero.Fs, t *template.Template, funcMap template.FuncMap) error {

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// List all files in the target directory
	files, err := afero.ReadDir(filesystem, ".")

	if err != nil {
		return derp.Wrap(err, "whisper.service.loadHTMLTemplateFromFilesystem", "Unable to list directory")
	}

	for _, file := range files {

		filename := file.Name()
		actionID, extension := list.SplitTail(filename, ".")

		// Only HTML files beyond this point...
		if extension == "html" {

			// Try to read the file from the filesystem
			content, err := afero.ReadFile(filesystem, filename)

			if err != nil {
				return derp.Report(derp.Wrap(err, "whisper.service.loadHTMLTemplateFromFilesystem", "Cannot read file", filename))
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Funcs(funcMap).Parse(contentString)

			if err != nil {
				return derp.Report(derp.Wrap(err, "whisper.service.loadHTMLTemplateFromFilesystem", "Unable to parse template HTML", contentString))
			}

			// Add this minified template into the resulting parse-tree
			t.AddParseTree(actionID, contentTemplate.Tree)
		}
	}

	// Return to caller.
	return nil
}

// loadModelFromFilesystem locates and parses a schema from the filesystem path
func loadModelFromFilesystem(filesystem afero.Fs, model interface{}) error {

	file, err := filesystem.Open("schema.json")

	if err != nil {
		return derp.Wrap(err, "whisper.service.loadModelFromFilesystem", "Cannot read file: schema.json")
	}

	// Load the file from the filesystem
	content, err := io.ReadAll(file)

	if err != nil {
		return derp.Wrap(err, "whisper.service.loadModelFromFilesystem", "Cannot read file: schema.json")
	}

	// Unmarshal the file into the schema.
	if err := json.Unmarshal(content, model); err != nil {
		return derp.Wrap(err, "whisper.service.loadModelFromFilesystem", "Invalid JSON configuration file: schema.json")
	}

	// Return to caller.
	return nil
}
