package service

import (
	"encoding/json"
	"html/template"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/benpate/schema"
	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// loadTemplateFromFilesystem locates and parses a Template sub-directory within the filesystem path
func loadTemplateFromFilesystem(directory string, funcMap template.FuncMap, t *template.Template) error {

	// Create a temporary value to collect .html files in.
	result := template.New("").Funcs(funcMap)

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// List all files in the target directory
	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return derp.Wrap(err, "ghost.service.loadFromFilesystem", "Unable to list directory", directory)
	}

	for _, file := range files {

		filename := file.Name()
		actionID, extension := list.SplitTail(filename, ".")

		// Only HTML files beyond this point...
		if extension == "html" {

			// Try to read the file from the filesystem
			content, err := ioutil.ReadFile(directory + "/" + filename)

			if err != nil {
				return derp.Report(derp.Wrap(err, "ghost.service.loadFromFilesystem", "Cannot read file", filename))
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Funcs(funcMap).Parse(contentString)

			if err != nil {
				return derp.Report(derp.Wrap(err, "ghost.service.loadFromFilesystem", "Unable to parse template HTML", contentString))
			}

			// Add this minified template into the resulting parse-tree
			result.AddParseTree(actionID, contentTemplate.Tree)
		}
	}

	// Copy the finalized result to the output
	t = result

	// Return to caller.
	return nil
}

// loadSchemaFromFilesystem locates and parses a schema from the filesystem path
func loadSchemaFromFilesystem(directory string, s *schema.Schema) error {

	// Load the file from the filesystem
	content, err := ioutil.ReadFile(directory + "/schema.json")

	if err != nil {
		return derp.Wrap(err, "ghost.service.loadFromFilesystem", "Cannot read file: schema.json", directory)
	}

	// Unmarshal the file into the schema.
	if err := json.Unmarshal(content, s); err != nil {
		return derp.Wrap(err, "ghost.service.loadFromFilesystem", "Invalid JSON configuration file: schema.json", directory)
	}

	// Return to caller.
	return nil
}
