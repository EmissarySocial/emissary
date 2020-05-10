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
	}

	result, err := cache.Render(data)

	spew.Dump(data)
	spew.Dump(result)
	spew.Dump(err)

	t.Error()
}

func populateTestTemplates(service Template) {

	t1 := model.Template{
		TemplateID: primitive.NewObjectID(),
		Format:     "ARTICLE",
		Content:    `<article><<h3>{{.title}}</h3><div>{{.body}}</div></article>`,
	}

	service.Save(&t1, "created")
}
