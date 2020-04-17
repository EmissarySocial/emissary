package content

import (
	"bytes"
	"html/template"
)

// TabTemplate represents the HTML that is returned for a Tab content object
var TabTemplate *template.Template

// TabFormatTabs represents a tab control with traditional tabs above the swappable content
const TabFormatTabs = "TABS"

// TabFormatButtons represents a tab control with buttons above the swappable content
const TabFormatButtons = "BUTTONS"

// TabFormatSidebar represents a tab control with a sidebar to the left of the swappable content
const TabFormatSidebar = "SIDEBAR"

func init() {
	TabTemplate = template.Must(template.New("TabTemplate").Parse(`<tab-group format="{{.Format}}"><tab-labels>{{range .Labels}}<tab-label>{{.}}</tab-label>{{end}}</tab-labels><tab-sections>{{range .Sections}}<tab-section>{{.HTML}}</tab-section>{{end}}</tab-sections></tab-group>`))
}

// Tab represents a list of content sections that can be selected by a "tab" control on the top, or left side of the screen.
type Tab struct {
	Format   string
	Labels   []string
	Sections []List
}

// HTML renders the tabs as a series of HTML elements
func (tab Tab) HTML() string {

	buffer := bytes.Buffer{}

	if err := TabTemplate.Execute(&buffer, tab); err != nil {
		return ""
	}

	return buffer.String()
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (tab Tab) WebComponents(accumulator map[string]bool) {

	accumulator["/components/content-tabs.js"] = true

	for _, content := range tab.Sections {
		content.WebComponents(accumulator)
	}
}
