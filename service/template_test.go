package service

import (
	"testing"

	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTemplate(t *testing.T) {

	// Load test environment
	factory := getTestFactory()

	template := factory.Template()
	populateTestTemplates(template)

	// Create Cache service
	cache := factory.TemplateCache()

	data := map[string]interface{}{
		"class": "ARTICLE",
		"title": "My Title",
		"body":  "My Body",
		"persons": []map[string]interface{}{
			{
				"name":  "John",
				"email": "john@connor.com",
			}, {
				"name":  "Sarah",
				"email": "sarah@sky.net",
			}, {
				"name":  "Kyle",
				"email": "kyle@resistance.mil",
			},
		},
	}

	result, err := cache.Render(data)

	// spew.Dump(data)
	spew.Dump(result)
	spew.Dump(err)

	t.Error()
}

func populateTestTemplates(service Template) {

	schema, err := populateSchema(`{
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
	`)

	if err != nil {
		panic(err)
	}

	t1 := model.Template{
		TemplateID: primitive.NewObjectID(),
		Format:     "ARTICLE",
		Content:    `{{define "person"}}<item><div>name: {{.name}}</div><div>{{.email}}</div></item>{{end -}}<article><h3>{{.title}}</h3><div>{{.body}}</div>{{range .persons}}{{template "person" .}}{{end}}</article>`,
		Schema:     *schema,
	}

	service.Save(&t1, "created")
}
