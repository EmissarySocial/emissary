package templatesource

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {

	source := NewFile("test")

	template, err := source.Load("simple")

	assert.Nil(t, err)
	assert.NotNil(t, template)

	derp.Report(err)
}
