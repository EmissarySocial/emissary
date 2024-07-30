package service

import (
	"html/template"
	"io/fs"
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ParseProfileURL(value string) (urlValue *url.URL, userID primitive.ObjectID, objectType string, objectID primitive.ObjectID, err error) {

	// Parse the value into a URL
	urlValue, err = url.Parse(value)

	if err != nil {
		return nil, primitive.NilObjectID, "", primitive.NilObjectID, derp.Wrap(err, "service.ParseURL", "Cannot parse value", value)
	}

	// View the path as a "/" delimited list
	path := list.BySlash(urlValue.Path).Tail()

	// First item in the path must be @<objectID>
	// Strip the "@" sign and convert to a primitive.ObjectID
	username, path := path.Split()

	if !strings.HasPrefix(username, "@") {
		return urlValue, primitive.NilObjectID, "", primitive.NilObjectID, derp.NewBadRequestError("service.ParseURL", "Username must begin with '@'", value)
	}

	username = strings.TrimPrefix(username, "@")

	userID, err = primitive.ObjectIDFromHex(username)

	if err != nil {
		return urlValue, userID, "", primitive.NilObjectID, derp.NewBadRequestError("service.ParseURL", "Username must be a valid hex string", value)
	}

	// If this is the end of the path, then we're done.
	if path.IsEmpty() {
		return urlValue, userID, "", primitive.NilObjectID, nil
	}

	// Second item in the path must be "/pub/"
	pub, path := path.Split()

	if pub != "pub" {
		return urlValue, userID, "", primitive.NilObjectID, derp.NewBadRequestError("service.ParseURL", "Path must contain with '/pub/'", value)
	}

	if path.IsEmpty() {
		return urlValue, userID, "", primitive.NilObjectID, nil
	}

	// Third item in the path must be the object type (followers, following, rules, etc.)
	objectType, path = path.Split()

	if path.IsEmpty() {
		return urlValue, userID, objectType, primitive.NilObjectID, nil
	}

	// Fourth item in the path must be the objectID for the underlying record.
	// Extract it and convert to a primitive.ObjectID
	objectName, path := path.Split()

	objectID, err = primitive.ObjectIDFromHex(objectName)

	if err != nil {
		return urlValue, userID, objectType, primitive.NilObjectID, derp.NewBadRequestError("service.ParseURL", "ObjectID must be a valid hex string", value)
	}

	// This should be the end of the path.
	if path.IsEmpty() {
		return urlValue, userID, objectType, objectID, nil
	}

	// But if we get here, then there are unrecognized values in the path.
	return urlValue, userID, objectType, objectID, derp.NewBadRequestError("service.ParseURL", "Path contains unrecognized values", value)
}

func ParseProfileURL_UserID(value string) (primitive.ObjectID, error) {
	_, userID, _, _, err := ParseProfileURL(value)
	return userID, err
}

func ParseProfileURL_AsFollowing(value string) (primitive.ObjectID, primitive.ObjectID, error) {
	_, userID, objectType, objectID, err := ParseProfileURL(value)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, "service.ParseProfileURL_AsFollowing", "Unable to parse profile URL", value)
	}

	if objectType != "following" {
		return primitive.NilObjectID, primitive.NilObjectID, derp.NewBadRequestError("service.ParseProfileURL_AsFollowing", "URL does not contain a following relationship", value)
	}

	return userID, objectID, nil
}

// notDeleted ensures that a criteria expression does not include soft-deleted items.
func notDeleted(criteria exp.Expression) exp.Expression {
	return criteria.And(exp.Equal("deleteDate", 0))
}

// loadHTMLTemplateFromFilesystem locates and parses a Template sub-directory within the filesystem path
func loadHTMLTemplateFromFilesystem(filesystem fs.FS, t *template.Template, funcMap template.FuncMap) error {

	// Create the minifier
	m := minify.New()
	minifier := html.Minifier{
		KeepEndTags:      true,
		KeepQuotes:       true,
		KeepDocumentTags: true,
	}

	m.AddFunc("text/html", minifier.Minify)

	// List all files in the target directory
	files, err := fs.ReadDir(filesystem, ".")

	if err != nil {
		return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to list directory", filesystem)
	}

	for _, file := range files {

		filename := file.Name()
		actionBytes, extension := list.Dot(filename).SplitTail()
		actionID := actionBytes.String()

		// Only HTML files beyond this point...
		if extension == "html" {

			// Try to read the file from the filesystem
			content, err := fs.ReadFile(filesystem, filename)

			if err != nil {
				return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Cannot read file", filename)
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Funcs(funcMap).Parse(contentString)

			if err != nil {
				return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to parse template HTML", contentString)
			}

			// Add this minified template into the resulting parse-tree
			if _, err := t.AddParseTree(actionID, contentTemplate.Tree); err != nil {
				derp.Report(derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to add template to parse tree", actionID))
			}
		}
	}

	// Return to caller.
	return nil
}

// DefinitionEmail marks a filesystem that contains an Email definition.
const DefinitionEmail = "EMAIL"

// DefinitionRegistration marks a filesystem that contains a Registration process definition
const DefinitionRegistration = "REGISTRATION"

// DefinitionEmail marks a filesystem that contains a stream Template definition.
const DefinitionTemplate = "TEMPLATE"

// DefinitionEmail marks a filesystem that contains a domain Theme definition.
const DefinitionTheme = "THEME"

// DefinitionWidget marks a filesystem that contains a Widget definition.
const DefinitionWidget = "WIDGET"

// findDefinition locates a definition JSON file and returns its type and contents
func findDefinition(filesystem fs.FS) (string, []byte, error) {

	// If this directory contains a "theme.json" file, then it's a theme.
	if file, err := readJSON(filesystem, "theme"); err == nil {
		return DefinitionTheme, file, nil
	}

	// If this directory contains a "template.json" file, then it's a template.
	if file, err := readJSON(filesystem, "template"); err == nil {
		return DefinitionTemplate, file, nil
	}

	// If this directory contains a "widget.json" file, then it's a widget.
	if file, err := readJSON(filesystem, "widget"); err == nil {
		return DefinitionWidget, file, nil
	}

	// If this directory contains a "register.json" file, then it's a registration form.
	if file, err := readJSON(filesystem, "registration"); err == nil {
		return DefinitionRegistration, file, nil
	}

	// If this directory contains an "email.json" file, then it's an email.
	if file, err := readJSON(filesystem, "email"); err == nil {
		return DefinitionEmail, file, nil
	}

	// TODO: LOW: Add DefinitionEmail to this.  Will need a *.json file in the email directory.

	return "", nil, derp.NewInternalError("service.findDefinition", "No definition file found")
}

// readJSON looks for JSON and HJSON files.
func readJSON(filesystem fs.FS, filename string) ([]byte, error) {

	if file, err := fs.ReadFile(filesystem, filename+".hjson"); err == nil {
		return file, nil
	}

	return fs.ReadFile(filesystem, filename+".json")
}

// pointerTo returns a pointer to a given value.  This is just
// some syntactic sugar for optional fields in API calls.
func pointerTo[T any](value T) *T {
	return &value
}

func slicesAreEqual(value1 []mapof.String, value2 []mapof.String) bool {
	// Lengths must be identical
	if len(value1) != len(value2) {
		return false
	}

	// Items at each index must be identical
	for index := range value1 {
		if !value1[index].Equal(value2[index]) {
			return false
		}
	}

	return true
}

// must strips out an error from a multi-result function call.
// This should be used sparingly because, while it does REPORT
// the error, it does not return it to the caller.
//
//lint:ignore U1000 This function is used in other packages.
func must[T any](value T, err error) T {
	if err != nil {
		derp.Report(err)
	}

	return value
}

// iif is a simple inline-if function.  You should probably never
// do something like this.  But fuck it.
func iif(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}
