package reader

// Context identifies the context within which the object exists or an activity was performed.
//
// The notion of "context" used is intentionally vague. The intended function is to serve as a means of grouping objects and activities that share a common originating context or purpose. An example could be all activities relating to a common project or event.
type Context struct {
	Vocabulary  string            `json:"vocabulary"`  // The primary vocabulary represented by the context/document.
	Language    string            `json:"language"`    // The language
	Extensions  map[string]string `json:"extensions"`  // a map of additional namespaces that are included in this context/document.
	NextContext *Context          `json:"nextContext"` // NextContext allows multiple contexts to be defined as a linked list.
}
