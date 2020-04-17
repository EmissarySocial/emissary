package content

// List represents several content items all strung together.  Most web pages are Lists
type List []Content

// Parse attempts to parse any data value into a List object
func (list *List) Parse(data interface{}) bool {

	switch data := data.(type) {
	case map[string]interface{}:

		result, success := Parse(data)

		if success {
			*list = append(*list, result)
		}

	case []interface{}:

		atLeastOneSuccess := false

		for _, record := range data {

			if result, success := Parse(record); success {
				*list = append(*list, result)
				atLeastOneSuccess = true
			}
		}

		return atLeastOneSuccess
	}

	return false
}

// HTML returns the HTML representation of the entire list of Content objects.
func (list *List) HTML() string {

	var result string

	for _, item := range *list {
		result = result + item.HTML()
	}

	return result
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (list *List) WebComponents(accumulator map[string]bool) {

	for _, content := range *list {
		content.WebComponents(accumulator)
	}
}
