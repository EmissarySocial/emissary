package service

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

// TemplateCache service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type TemplateCache struct {
	Cache   map[string]model.Template
	FuncMap map[string]interface{}
}

// NewTemplateCache loads all templates into memory for the duration of the server.
func NewTemplateCache(factory Factory) (*TemplateCache, []*derp.Error) {

	var errors []*derp.Error
	var object model.Template

	funcMap := map[string]interface{}{
		"MyFunc": func(v string) string {
			return "MyFunc, MyFunc, My Lovely Little Func."
		},
	}

	templateCache := TemplateCache{
		Cache:   map[string]model.Template{},
		FuncMap: funcMap,
	}

	// Get TemplateService to load ALL templates from the database
	service := factory.Template()

	// Load all Templates from the database
	it, err := service.List(nil)

	if err != nil {
		errors = append(errors, derp.Wrap(err, "service.TemplateCache.Init", "Failed Loading Template Cache"))
		return nil, errors
	}

	// For each object in the result set
	for it.Next(&object) {

		err := object.Init(funcMap)

		// Report errors, if necessary
		if err != nil {
			errors = append(errors, derp.Wrap(err, "service.TemplateCache.Init", "Error Compiling Template Content", object.Format))
			continue // One broken template should not stop the whole server.
		}

		// Add the object to the cache.
		templateCache.Cache[object.Format] = object
	}

	// Success!
	return &templateCache, errors
}

func (cache *TemplateCache) Render(data map[string]interface{}) (string, *derp.Error) {

	var result bytes.Buffer

	template, err := cache.GetTemplate(data)

	if err != nil {
		return "", derp.Wrap(err, "service.TemplateCache.Render", "Could not load template for data", data)
	}

	if err := template.Compiled.Execute(&result, data); err != nil {
		return "", derp.New(500, "service.TemplateCache.Render", "Could not execute template", err)
	}

	return result.String(), nil
}

func (cache *TemplateCache) GetTemplate(data map[string]interface{}) (*model.Template, *derp.Error) {

	if class, ok := data["class"]; ok {

		if classString, ok := class.(string); ok {

			if template, ok := cache.Cache[classString]; ok {
				return &template, nil
			}

			return nil, derp.New(500, "service.TemplateCache.Render", "Template does not exist in cache", classString)
		}
		return nil, derp.New(500, "service.TemplateCache.Render", "Invalid class.  Class must be a string", class)
	}
	return nil, derp.New(500, "service.TemplateCache.Reader", "Invalid data.  Missing 'class' string")
}
