package writer

type Collection struct {
	Context    *Context `json:"@context,omitempty"`
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Type       []string `json:"type,omitempty"`
	TotalItems int      `json:"totalItems,omitempty"` // A non-negative integer specifying the total number of objects contained by the logical view of the collection. This number might not reflect the actual number of items serialized within the Collection object instance.
	Current    string   `json:"current,omitempty"`    // In a paged Collection, indicates the page that contains the most recently updated member items.
	First      string   `json:"first,omitempty"`      // In a paged Collection, indicates the furthest preceeding page of items in the collection.
	Last       string   `json:"last,omitempty"`       // In a paged Collection, indicates the furthest proceeding page of the collection.
	Items      []Object `json:"items,omitempty"`
	Ordered    bool
}

/*
func (collection *Collection) MarshalJSON() ([]byte, error) {

	var result bytes.Buffer

	result.WriteRune('{')

	if collection.Context != nil {
		result.WriteString(`"@context":`)
		result.Write(toJSON(collection.Context))
		result.WriteRune(',')
	}

	if collection.Name != "" {
		result.WriteString(`"id":`)
		// result.Write()
	}
	result.WriteRune('}')

	return result.Bytes(), nil
}
*/
