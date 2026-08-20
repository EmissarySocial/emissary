package build

import (
	"io"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepMarkFolderRead is a Step that marks every unread NewsItem in a folder (identified by
// the "folderId" query parameter) as read.
type StepMarkFolderRead struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepMarkFolderRead) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post marks every unread NewsItem in the requested folder as read.
func (step StepMarkFolderRead) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepMarkFolderRead.Post"

	// RULE: User MUST be authenticated to use this step
	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.Unauthorized(location, "Anonymous user is not authorized to perform this action"))
	}

	// A missing/invalid folderId (e.g. the "News Feed" all-folders view) is a no-op, not an error.
	folderID, err := primitive.ObjectIDFromHex(builder.QueryParam("folderId"))

	if err != nil || folderID.IsZero() {
		return Continue()
	}

	newsFeedService := builder.factory().NewsFeed()

	if err := newsFeedService.MarkAllReadByFolder(builder.session(), builder.AuthenticatedID(), folderID); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Marking folder read", folderID))
	}

	return Continue()
}
