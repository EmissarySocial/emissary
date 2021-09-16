package writer

// Context identifies the context within which the object exists or an activity was performed.
//
// The notion of "context" used is intentionally vague. The intended function is to serve as a means of grouping objects and activities that share a common originating context or purpose. An example could be all activities relating to a common project or event.
type Context struct {
	Vocabulary  string            // The primary vocabulary represented by the context/document.
	Language    string            // The language
	Extensions  map[string]string // a map of additional namespaces that are included in this context/document.
	NextContext *Context          // NextContext allows multiple contexts to be defined as a linked list.
}

/*
// NewContext represents the standard context defined by the W3C
func NewContext(vocabulary string, language string) *Context {
	return &Context{
		Vocabulary: vocabulary,
		Language:   language,
		Extensions: map[string]string{},
	}
}

// DefaultContext represents the standard context defined by the W3C
func DefaultContext() *Context {
	return &Context{
		Vocabulary: "https://www.w3.org/ns/activitystreams",
		Language:   "und",
	}
}

// Extension safely adds an extension record to this context
func (c *Context) Extension(key string, value string) {

	if c.Extensions == nil {
		c.Extensions = map[string]string{}
	}

	c.Extensions[key] = value
}

// MarshalJSON implements the Marshaller interface, and allows
// Context objects to be represented according to the ActivityStream
// standard.
func (c *Context) MarshalJSON() ([]byte, error) {

	// In most cases, we're only using a single context, so
	// we'll only have to marshal a single string/object.
	if c.NextContext == nil {
		return c.ToJSON(), nil
	}

	// Fall through to here means that we have >1 Context,
	// so it needs to be represented as an array.
	var result bytes.Buffer

	result.WriteRune('[')
	for true {
		result.Write(c.ToJSON())

		if c.NextContext == nil {
			break
		} else {
			c = c.NextContext
			result.WriteRune(',')
		}
	}
	result.WriteRune(']')
	return result.Bytes(), nil
}

// UnmarshalJSON implements the Unmarshaller interface, and allows
// Context objects to be read from external ActivityStream sources.
func (c *Context) UnmarshalJSON([]byte) error {
	return nil
}

// ToJSON returns the JSON string for just this one Context --
// without checking to see if the "NextItem" is populated.
func (c *Context) ToJSON() []byte {

	if (c.Language == "und") || (c.Language == "") && (len(c.Extensions) == 0) {
		result, _ := json.Marshal(c.Vocabulary)
		return result
	}

	var result bytes.Buffer

	result.WriteString(`{"@vocab":`)
	result.Write(toJSON(c.Vocabulary))

	if c.Language != "" {
		result.WriteString(`,"@language":`)
		result.Write(toJSON(c.Language))
	}

	for key, value := range c.Extensions {
		result.WriteString(`,"` + key + `":`)
		result.Write(toJSON(value))
	}

	result.WriteString("}")

	return result.Bytes()
}
*/
