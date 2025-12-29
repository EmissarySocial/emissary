package service

import (
	"context"
	"html/template"
	"io/fs"
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/dlclark/metaphone3"
	"github.com/rs/zerolog"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RangeFunc converts a data.Iterator into a Go 1.23 RangeFunc (https://go.dev/blog/range-functions)
// Deprecated: This should be coded directly into each service.
func RangeFunc[T any](it data.Iterator, new func() T) iter.Seq[T] {

	return func(yield func(T) bool) {

		defer derp.ReportFunc(it.Close)

		if it == nil {
			return
		}

		for value := new(); it.Next(&value); value = new() {
			if !yield(value) {
				break
			}
		}
	}
}

func joinIterators[T any](iterators ...iter.Seq[T]) iter.Seq[T] {

	return func(yield func(T) bool) {
		for _, iterator := range iterators {
			for value := range iterator {
				if !yield(value) {
					return
				}
			}
		}
	}
}

func iterateFollowerAddresses(followers iter.Seq[model.Follower]) iter.Seq[string] {

	return func(yield func(string) bool) {
		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return
			}
		}
	}
}

func textIndex(tokens ...string) sliceof.String {

	// RULE: Exit early on empty strings (why would you do this? Who hurt you?)
	if len(tokens) == 0 {
		return make(sliceof.String, 0)
	}

	// Split the value into words (strips hashtags and special characters)
	result := make([]string, 0, len(tokens))
	encoder := metaphone3.Encoder{}

	for _, token := range tokens {
		token, _ = encoder.Encode(token)
		if token != "" {
			result = append(result, token)
		}
	}

	return result
}

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
		return urlValue, primitive.NilObjectID, "", primitive.NilObjectID, derp.BadRequestError("service.ParseURL", "Username must begin with '@'", value)
	}

	username = strings.TrimPrefix(username, "@")

	userID, err = primitive.ObjectIDFromHex(username)

	if err != nil {
		return urlValue, userID, "", primitive.NilObjectID, derp.BadRequestError("service.ParseURL", "Username must be a valid hex string", value)
	}

	// If this is the end of the path, then we're done.
	if path.IsEmpty() {
		return urlValue, userID, "", primitive.NilObjectID, nil
	}

	// Second item in the path must be "/pub/"
	pub, path := path.Split()

	if pub != "pub" {
		return urlValue, userID, "", primitive.NilObjectID, derp.BadRequestError("service.ParseURL", "Path must contain with '/pub/'", value)
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
		return urlValue, userID, objectType, primitive.NilObjectID, derp.BadRequestError("service.ParseURL", "ObjectID must be a valid hex string", value)
	}

	// This should be the end of the path.
	if path.IsEmpty() {
		return urlValue, userID, objectType, objectID, nil
	}

	// But if we get here, then there are unrecognized values in the path.
	return urlValue, userID, objectType, objectID, derp.BadRequestError("service.ParseURL", "Path contains unrecognized values", value)
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
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequestError("service.ParseProfileURL_AsFollowing", "URL does not contain a following relationship", value)
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
		actionBytes, extension := list.Dot(filename).SplitTail() // nolint:scopeguard (readability)

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
			actionID := actionBytes.String()
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

// DefinitionNotFound is returned when no definition file exists in the directory.
const DefinitionNotFound = "NOT-FOUND"

// findDefinition locates a definition JSON file and returns its type and contents
func findDefinition(filesystem fs.FS) (string, []byte) {

	// If this directory contains a "theme.json" file, then it's a theme.
	if file, err := readJSON(filesystem, "theme"); err == nil {
		return DefinitionTheme, file
	}

	// If this directory contains a "template.json" file, then it's a template.
	if file, err := readJSON(filesystem, "template"); err == nil {
		return DefinitionTemplate, file
	}

	// If this directory contains a "widget.json" file, then it's a widget.
	if file, err := readJSON(filesystem, "widget"); err == nil {
		return DefinitionWidget, file
	}

	// If this directory contains a "register.json" file, then it's a registration form.
	if file, err := readJSON(filesystem, "registration"); err == nil {
		return DefinitionRegistration, file
	}

	// If this directory contains an "email.json" file, then it's an email.
	if file, err := readJSON(filesystem, "email"); err == nil {
		return DefinitionEmail, file
	}

	// TODO: LOW: Add DefinitionEmail to this.  Will need a *.json file in the email directory.

	return DefinitionNotFound, nil
}

// readJSON looks for JSON and HJSON files.
func readJSON(filesystem fs.FS, filename string) ([]byte, error) {

	if file, err := fs.ReadFile(filesystem, filename+".hjson"); err == nil {
		return file, nil
	}

	return fs.ReadFile(filesystem, filename+".json")
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

// firstOf is a quickie generic helper that returns the first
// non-zero value from a list of comparable values.
func firstOf[T comparable](values ...T) T {

	var empty T

	// Try each value in the list.  If non-zero, then celebrate success.
	for _, value := range values {
		if value != empty {
			return value
		}
	}

	// Boo, hisss...
	return empty
}

// pointerTo returns a pointer to a given value.  This is just
// some syntactic sugar for optional fields in API calls.
func pointerTo[T any](value T) *T {
	return &value
}

// mapProductsToLookupCodes converts a slice of Products into a slice of LookupCodes
func mapProductsToLookupCodes(remoteProducts ...model.Product) sliceof.Object[form.LookupCode] {

	result := make(sliceof.Object[form.LookupCode], len(remoteProducts))

	for index, remoteProduct := range remoteProducts {
		result[index] = remoteProduct.LookupCode()
	}

	return result
}

// flatten converts a map of slices into a single slice
func flatten(original mapof.Object[id.Slice]) id.Slice {

	length := len(original)

	if length == 0 {
		return id.Slice{}
	}

	result := make(id.Slice, 0, length)

	for _, value := range original {
		result = append(result, value...)
	}

	return result
}

// timeoutContext creates a context with a timeout in seconds
func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

// iif is a generic inline-if helper function
func iif[T any](condition bool, trueValue T, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// canTrace returns TRUE if zerolog is configured to allow Trace logs
// nolint:unused
func canTrace() bool {
	return canLog(zerolog.TraceLevel)
}

// canLog is a silly zerolog helper that returns TRUE
// if the provided log level would be allowed
// (based on the global log level).
// This makes it easier to execute expensive code conditionally,
// for instance: marshalling a JSON object for logging.
func canLog(level zerolog.Level) bool {
	return zerolog.GlobalLevel() <= level
}
