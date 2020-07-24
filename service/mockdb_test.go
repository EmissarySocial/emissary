package service

import (
	"testing"

	"github.com/benpate/data/expression"
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

	result, err := service.Load(expression.New("token", "=", "1-my-first-stream"))

	assert.Nil(t, err)
	spew.Dump(result)
}
