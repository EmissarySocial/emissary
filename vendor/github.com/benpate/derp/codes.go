package derp

// Derp recommends, but does not require, using HTTP status codes as error messages.
// Several of the most useful messages are listed here as defaults.

const (

	// CodeBadRequestError indicates that the request is not properly formatted.
	CodeBadRequestError = 400

	// CodeForbiddenError means that the current user does not have the required permissions to access the requested resource.
	CodeForbiddenError = 403

	// CodeNotFoundError represents a request for a resource that does not exist, such as a database query that returns "not found"
	CodeNotFoundError = 404

	// CodeInternalError represents a generic error message, given when an unexpected condition was encountered and no more specific message is suitable.
	CodeInternalError = 500
)
