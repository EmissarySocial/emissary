package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service/templateSource"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestLoadTemplates(t *testing.T) {

	// Load test environment
	factory := getTestFactory()

	templateService := factory.Template()
	populateTestTemplates(templateService)

	spew.Dump(templateService)
}

func TestTemplate(t *testing.T) {

	// Load test environment
	factory := getTestFactory()

	templateService := factory.Template()
	populateTestTemplates(templateService)

	streamService := factory.Stream()
	populateTestStreamService(streamService)

	stream, err := streamService.LoadByToken("1-my-first-stream")
	assert.Nil(t, err)

	html, err := streamService.Render(stream, "default")

	assert.Nil(t, err)
	derp.Report(err)
	t.Log(html)

}

func populateTestTemplates(service *Template) {

	testTemplates := templateSource.NewFile("templateSource/test")

	{
		simple, err := testTemplates.Load("simple")
		service.Save(simple, "loaded from filesystem")
		derp.Report(err)
	}

	{
		article, err := testTemplates.Load("article")
		service.Save(article, "loaded from filesystem")
		derp.Report(err)
	}

	/*
		v1 := `<article><h3>{{.Label}}</h3><div>{{.Description}}</div>{{range .Data.persons}}<item><div>name: {{.name}}</div><div>{{.email}}</div></item>{{end}}</article>`

		t1 := model.Template{
			TemplateID: "ARTICLE",
			Views: map[string]model.View{
				"default": {
					Label: "Default",
					HTML:  v1,
				},
			},
		}

		t1.Schema, _ = schema.UnmarshalJSON([]byte(`{
			"url": "example.com/test-template",
			"title": "Test Template Schema",
			"type": "object",
			"properties": {
				"class": {
					"type": "string"
				},
				"title": {
					"type": "string",
					"description": "The human-readable title for this article"
				},
				"body": {
					"type": "string",
					"description": "The HTML content for this article"
				},
				"persons": {
					"description": "Array of people to render on the page",
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"email": {
								"type":"string"
							}
						}
					}
				},
				"friends": {
				  "type" : "array",
				  "items" : { "title" : "REFERENCE", "$ref" : "#" }
				}
			},
			"required": ["class", "title", "body", "persons"]
		  }
		`))

		service.Save(&t1, "created")
	*/
}
