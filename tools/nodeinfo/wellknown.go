package nodeinfo

// Wellknown is a struct that represents the .well-known nodeinfo endpoint
type Wellknown struct {
	Links []Link `json:"links"`
}

// Link represents an individual link in the .well-known nodeinfo endpoint
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// NewWellknown returns a fully initialized Wellknown object
func NewWellknown() Wellknown {
	return Wellknown{
		Links: make([]Link, 0),
	}
}
