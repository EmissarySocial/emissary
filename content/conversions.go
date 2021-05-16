package content

import "github.com/benpate/datatype"

func FromHTML(html string) Content {
	return Content{
		{
			Type: "HTML",
			Data: datatype.Map{
				"html": html,
			},
		},
	}
}
