package writer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {

	// Test default context data
	{
		c := DefaultContext()

		assert.Equal(t, "https://www.w3.org/ns/activitystreams", c.Vocabulary)
		assert.Equal(t, "und", c.Language)
		assert.Zero(t, len(c.Extensions))
		assert.Nil(t, c.NextContext)

		assert.Equal(t, `"https://www.w3.org/ns/activitystreams"`, string(c.ToJSON()))
	}

	// Test custom context, and chaining multiple contexts
	{
		c := NewContext("https://test.com", "en-us")

		assert.Equal(t, "https://test.com", c.Vocabulary)
		assert.Equal(t, "en-us", c.Language)
		assert.Zero(t, len(c.Extensions))
		assert.Nil(t, c.NextContext)

		assert.Equal(t, `{"@vocab":"https://test.com","@language":"en-us"}`, string(c.ToJSON()))

		c.Extension("ext", "https://extension.com/ns/activitystreams")

		assert.Equal(t, `{"@vocab":"https://test.com","@language":"en-us","ext":"https://extension.com/ns/activitystreams"}`, string(c.ToJSON()))

		json1, err1 := c.MarshalJSON()

		assert.Equal(t, json1, c.ToJSON())
		assert.Nil(t, err1)

		c.NextContext = DefaultContext()

		json2, err2 := c.MarshalJSON()

		assert.Equal(t, `[{"@vocab":"https://test.com","@language":"en-us","ext":"https://extension.com/ns/activitystreams"},"https://www.w3.org/ns/activitystreams"]`, string(json2))
		assert.Nil(t, err2)
	}

	// Test safely adding an extension to an improperly initialized context
	{
		c := Context{Vocabulary: "https://test.com"}
		c.Extension("dog", "https://dog.com/ns/activitystreams")

		assert.Equal(t, "https://test.com", c.Vocabulary)
		assert.Equal(t, "", c.Language)
		assert.Equal(t, c.Extensions["dog"], "https://dog.com/ns/activitystreams")
	}
}
