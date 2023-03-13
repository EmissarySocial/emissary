package sherlock

type Page struct {
	CanonicalURL string
	Type         string
	Title        string
	Description  string
	Image        Image
	ProviderName string
	ProviderURL  string
	PublishDate  int64
	Authors      []Author
	Tags         []string
	InReplyTo    string
	Locale       string
}

type Author struct {
	Name  string
	URL   string
	Email string
	Image Image
}

type Image struct {
	URL    string
	Height int
	Width  int
}

func NewPage() Page {
	return Page{
		Authors: make([]Author, 0),
		Tags:    make([]string, 0),
	}
}

// IsEmpty returns TRUE if there is no Title, Description, or Image URL
func (data Page) IsEmpty() bool {
	return (data.Title == "") && (data.Description == "") && (data.Image.URL == "")
}
