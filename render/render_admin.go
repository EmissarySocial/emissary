package render

import "github.com/benpate/ghost/model"

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func AdminSections() []model.Option {
	return []model.Option{
		{
			Value: "users",
			Label: "Users",
		},
		{
			Value: "groups",
			Label: "Groups",
		},
		{
			Value: "content",
			Label: "Content",
		},
	}
}
