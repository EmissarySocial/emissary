package domain

import (
	"testing"

	"github.com/benpate/exp"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestMockDB(t *testing.T) {

	factory := getTestFactory()

	service := factory.Stream()

	stream := service.New()

	stream.Token = "1-my-first-stream"

	err := service.Save(stream, "New Stream")
	assert.Nil(t, err)

	result, err := service.Load(exp.Equal("token", "1-my-first-stream"))

	assert.Nil(t, err)
	spew.Dump(result)
}
