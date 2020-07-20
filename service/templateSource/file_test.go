package templatesource

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {

	source := NewFile("test")

	template, err := source.Load("simple")

	assert.Nil(t, err)
	assert.NotNil(t, template)

	derp.Report(err)
	spew.Dump(template)
}
