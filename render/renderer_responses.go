package render

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Responses renderer is a lightweight renderer that contains the
// data and types required to render the responses widget.
type Responses struct {
	userID          primitive.ObjectID // The currently signed in user
	internalURL     string             // URL of the internal object (used for additional actions)
	objectID        string             // The ObjectID (URL) of the object that is being responded to
	responseService *service.Response  // A pointer to the Response Service, which will load other data from the database.
}

func NewResponses(userID primitive.ObjectID, internalURL string, objectID string, responseService *service.Response) Responses {
	return Responses{
		userID:          userID,
		internalURL:     internalURL,
		objectID:        objectID,
		responseService: responseService,
	}
}

// InternalURL returns the InternalURL of the object that is being rendered
func (w Responses) InternalURL() string {
	return w.internalURL
}

// ObjectID returns the ObjectID (URL) of the object that is being rendered
func (w Responses) ObjectID() string {
	return w.objectID
}

// UserID returns the unique id of the currently signed in user
func (w Responses) UserID() primitive.ObjectID {
	return w.userID
}

// CountByContent uses the responseService to count the number of responses that match each content value (👍, 👎, etc)
func (w Responses) CountByContent() (mapof.Int, error) {
	return w.responseService.CountByContent(w.objectID)
}

func (w Responses) UserResponse() (string, error) {

	// RULE: If the user is not signed in, then we don't need to look for "their" response.
	if w.userID.IsZero() {
		return "", nil
	}

	// Try to load the User's response
	response := model.NewResponse()
	err := w.responseService.LoadByUserAndObject(w.userID, w.objectID, &response)

	if err == nil {
		return response.Content, nil
	}

	if derp.NotFound(err) {
		return "", nil
	}

	return "", derp.Wrap(err, "render.Responses.UserResponses", "Error loading user response")
}
