package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestText(t *testing.T) {

	text1 := Text("This is a story of a man named brady")
	assert.Equal(t, "This is a story of a man named brady", text1.HTML())

	text2 := Text(`This is a story
of a man named brady.

Who was living with three boys of his own...`)

	assert.Equal(t, "This is a story<br>of a man named brady.<br><br>Who was living with three boys of his own...", text2.HTML())
}
