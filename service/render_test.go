package service

import (
	"testing"

	"github.com/benpate/ghost/model"
)

func TestRender(t *testing.T) {

	var stream *model.Stream

	factory := getTestFactory()

	r := factory.Render()

	pipeline := r.Pipeline(r.Stream(stream), r.Page(""))

	if true {
		pipeline = pipeline.Into(r.Global())
	}

	pipeline.Execute()
}
