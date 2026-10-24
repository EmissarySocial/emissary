package mastodon

import (
	"mime/multipart"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const profileImageMaxBytes = 8 << 20 // Largest avatar or header upload accepted

// profileImage describes one of the two images on a profile, filed the way the web profile editor files it.
type profileImage struct {
	category string // Attachment category
	width    int    // Fixed width the image is served at (0 keeps the original)
	height   int    // Fixed height the image is served at (0 keeps the original)
}

var (
	profileAvatar = profileImage{category: "icon", width: 400, height: 400} // a square avatar
	profileHeader = profileImage{category: "image"}                         // a banner of any size
)

// https://docs.joinmastodon.org/methods/profile/
func DeleteProfile_Avatar(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Avatar) (object.Account, error) {

	const location = "handler.mastodon.DeleteProfile_Avatar"

	return func(auth model.Authorization, t txn.DeleteProfile_Avatar) (object.Account, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Loading User", auth.UserID)
		}

		// Delete the user's Avatar
		if err := userService.DeleteAvatar(session, &user, "Deleted via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Deleting Avatar")
		}

		return tootCredentialUser(factory, session, auth, &user), nil
	}
}

// DeleteProfile_Header implements the Mastodon "delete profile header image" endpoint
func DeleteProfile_Header(serverFactory *server.Factory) func(model.Authorization, txn.DeleteProfile_Header) (object.Account, error) {

	const location = "handler.mastodon.DeleteProfile_Header"

	return func(auth model.Authorization, t txn.DeleteProfile_Header) (object.Account, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Invalid Domain Name", t.Host)
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Account{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the current User
		userService := factory.User()
		user := model.NewUser()

		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Loading User", auth.UserID)
		}

		// Delete the user's header image
		if err := userService.DeleteImage(session, &user, "Deleted via Mastodon API"); err != nil {
			return object.Account{}, derp.Wrap(err, location, "Deleting header")
		}

		return tootCredentialUser(factory, session, auth, &user), nil
	}
}

// saveProfileImage stores an uploaded image as the User's avatar or header, replacing the old one.
// slot is the User field that holds the attachment ID; the caller saves the User afterward.
func saveProfileImage(factory *service.Factory, session data.Session, user *model.User, file *multipart.FileHeader, kind profileImage, slot *primitive.ObjectID) error {

	const location = "handler.mastodon.saveProfileImage"

	// Open the uploaded file
	source, err := file.Open()

	if err != nil {
		return derp.Wrap(err, location, "Opening uploaded file")
	}

	defer derp.ReportFunc(source.Close)

	// Read the file's type and size
	reader, contentType, width, height, err := sniffMedia(source)

	if err != nil {
		return derp.Wrap(err, location, "Reading uploaded file")
	}

	// RULE: only an image of a reasonable size may be used
	if err := checkProfileImage(contentType, file.Size); err != nil {
		return derp.Wrap(err, location, "Rejected upload", file.Filename)
	}

	// Describe the new attachment
	attachment := model.NewAttachment(model.AttachmentObjectTypeUser, user.UserID)
	attachment.Original = file.Filename
	attachment.ContentType = contentType
	attachment.Category = kind.category
	attachment.Width = width
	attachment.Height = height
	attachment.SetRules(kind.width, kind.height, []string{"webp"})

	// Store the file
	if err := factory.MediaServer().Put(attachment.AttachmentID.Hex(), reader); err != nil {
		return derp.Wrap(err, location, "Saving uploaded file")
	}

	// Save the attachment record
	attachmentService := factory.Attachment()

	if err := attachmentService.Save(session, &attachment, "Uploaded via Mastodon API"); err != nil {
		return derp.Wrap(err, location, "Saving attachment record")
	}

	// Remove the previous image; a leftover file is not worth failing over
	if previous := *slot; !previous.IsZero() {

		if err := attachmentService.DeleteByID(session, model.AttachmentObjectTypeUser, user.UserID, previous, "Replaced via Mastodon API"); err != nil && !derp.IsNotFound(err) {
			derp.Report(derp.Wrap(err, location, "Removing previous image", previous))
		}
	}

	// Point the User at the new image
	*slot = attachment.AttachmentID
	return nil
}

// checkProfileImage confirms an upload is an image of a reasonable size.
func checkProfileImage(contentType string, size int64) error {

	const location = "handler.mastodon.checkProfileImage"

	if !strings.HasPrefix(contentType, "image/") {
		return derp.BadRequest(location, "Profile pictures must be images", contentType)
	}

	if size > profileImageMaxBytes {
		return derp.BadRequest(location, "Image is too large", size)
	}

	return nil
}
