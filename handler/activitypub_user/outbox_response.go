package activitypub_user

import "net/http"

// respondCreated answers a C2S POST with `201 Created`, and names the object that the activity
// wrote when there is one. The activity's own id travels in `X-Activity-Id`.
func respondCreated(context Context, objectURL string) error {

	// Name the object twice: `Location` is what clients read today, and `X-Object-Id` is where
	// the object's URL stays once `Location` carries the activity's id (BUG-57)
	if objectURL != "" {
		header := context.context.Response().Header()
		header.Set("Location", objectURL)
		header.Set("X-Object-Id", objectURL)
	}

	return context.context.NoContent(http.StatusCreated)
}
