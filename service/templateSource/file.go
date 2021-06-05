package templatesource

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/fsnotify/fsnotify"

	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// File is a TemplateSource adapter that can load/save Templates from/to the local filesytem.
type File struct {
	Path string
}

// NewFile returns a fully initialized File adapter for loading/saving Templates
func NewFile(path string) *File {
	return &File{
		Path: path,
	}
}

// List returns all Templates produced by this TemplateSource
func (fs *File) List() ([]string, error) {

	list, err := ioutil.ReadDir(fs.Path)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.templateSource.File.List", "Unable to list files in filesystem", fs)
	}

	result := make([]string, len(list))

	// Use a separate counter because not all files will be included in the result
	counter := 0
	for _, fileInfo := range list {

		if fileInfo.IsDir() {
			name := fileInfo.Name()
			if (name != "layout") && (name != "static") {
				fmt.Println(name)
				result[counter] = fileInfo.Name()
				counter = counter + 1
			}
		}
	}

	result = result[:counter]

	return result, nil
}

// Load tries to find a template sub-directory within the filesystem path
func (fs *File) Load(templateID string) (*model.Template, error) {

	result := model.NewTemplate(templateID)

	directory := fs.Path + "/" + templateID

	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Unable to list directory", directory)
	}

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// Load the schema.json file
	{
		content, err := ioutil.ReadFile(directory + "/schema.json")

		if err != nil {
			return nil, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Cannot read file: schema.json")
		}

		if err := json.Unmarshal(content, result); err != nil {
			return nil, derp.Wrap(err, "ghost.service.templateSource.File.Load", "Invalid JSON configuration file: schema.json")
		}
	}

	// Views are processed FIRST so we can generate a list of objects to enter into the different States
	for _, file := range files {

		filename := file.Name()
		actionID, extension := list.SplitTail(filename, ".")

		// Only HTML files beyond this point...
		switch extension {

		case "html":

			// Verify that the action is already defined in the schema.json
			if _, ok := result.Actions[actionID]; !ok {
				return nil, derp.New(derp.CodeInternalError, "ghost.service.templateSource.File.Load", "Missing action", actionID)
			}

			// Try to read the file from the filesystem
			content, err := ioutil.ReadFile(directory + "/" + filename)

			if err != nil {
				return nil, derp.Report(derp.Wrap(err, "ghost.service.templateSource.File.Load", "Cannot read file", filename))
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Parse(contentString)

			if err != nil {
				return nil, derp.Report(derp.Wrap(err, "ghost.model.View.Compiled", "Unable to parse template HTML", contentString))
			}

			// Put the parsed/minified template into the custom data of the action.
			result.Actions[actionID].Map["template"] = contentTemplate
		}
	}

	return result, nil
}

/////////////////////////////////////
/// REAL TIME UPDATES

// Watch populates a channel of model.Template objects every time a template is updated.
func (fs *File) Watch(updateChannel chan model.Template) error {

	fmt.Println("templateSource.Watch")
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "ghost.service.templateSource", "Could not watch filesystem")
	}

	files, err := ioutil.ReadDir(fs.Path)

	if err != nil {
		return derp.Wrap(err, "ghost.service.templateSource", "Could not read directory", fs.Path)
	}

	for _, file := range files {
		if file.IsDir() {
			if err := watcher.Add(fs.Path + "/" + file.Name()); err != nil {
				return derp.Wrap(err, "ghost.service.templateSource.File", "Error adding watcher on path", fs.Path, file)
			}
		}
	}

	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:

				fmt.Println("templateSource.Watch.receivedEvent")
				if ok {
					if event.Op != fsnotify.Write {
						continue
					}

					templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")

					template, err := fs.Load(templateName)

					if err != nil {
						derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error loading changes to template", event, templateName))
						continue
					}

					fmt.Println("templateSource.Watch.sendingTemplate: " + template.Label)
					updateChannel <- *template
				}

			case err, ok := <-watcher.Errors:

				if ok {
					derp.Report(derp.Wrap(err, "ghost.service.templateSource.File", "Error watching filesystem"))
				}
			}
		}
	}()

	return nil
}
