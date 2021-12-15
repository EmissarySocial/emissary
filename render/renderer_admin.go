package render

import "github.com/benpate/ghost/model"

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func AdminSections() []model.Option {
	return []model.Option{
		{
			Value: "domain",
			Label: "Site",
		},
		{
			Value: "content",
			Label: "Navigation",
		},
		{
			Value: "groups",
			Label: "Groups",
		},
		{
			Value: "users",
			Label: "People",
		},
	}
}
