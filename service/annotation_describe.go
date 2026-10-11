package service

import (
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// annotationMaxLength matches the MaxLength of the "name" and "icon" fields in model.AnnotationSchema
const annotationMaxLength = 256

// describeAnnotation copies the annotated document's name and icon onto the Annotation,
// and stamps its type when the Annotation is new
func describeAnnotation(client streams.Client, annotation *model.Annotation) error {

	const location = "service.describeAnnotation"

	document, err := client.Load(annotation.URL)

	if err != nil {
		return derp.Wrap(err, location, "Loading annotated document", annotation.URL)
	}

	// RULE: The type is decided once, when the Annotation is created, and never re-read
	if annotation.IsNew() {
		annotation.Type = annotationType(document)
	}

	annotation.Name = annotationName(document)
	annotation.Icon = annotationIcon(document)

	return nil
}

// annotationType returns ACTOR for an ActivityPub actor, and DOCUMENT for anything else
func annotationType(document streams.Document) string {

	if document.IsActor() {
		return model.AnnotationTypeActor
	}

	return model.AnnotationTypeDocument
}

// annotationName returns a readable name for an annotated document, falling back to its
// summary and then its content, because most posts have no name of their own.
func annotationName(document streams.Document) string {

	// Actors are named by their display name, then their username
	if document.IsActor() {
		return truncateRunes(firstNonEmpty(document.Name(), document.UsernameOrID()), annotationMaxLength)
	}

	// Documents fall back to a plain-text excerpt of their summary or content, with entities
	// decoded so that the template's own escaping does not show them as "&#39;"
	result := firstNonEmpty(
		document.Name(),
		plainText(document.Summary()),
		plainText(document.Content()),
	)

	return truncateRunes(result, annotationMaxLength)
}

// annotationIcon returns an icon URL for an annotated document, falling back to the
// author's icon, because most posts have none of their own.
func annotationIcon(document streams.Document) string {

	result := document.Icon().Href()

	if result == "" && !document.IsActor() {
		result = document.AttributedTo().Icon().Href()
	}

	// RULE: A URL cannot be shortened, so one too long to store is left out
	if utf8.RuneCountInString(result) > annotationMaxLength {
		return ""
	}

	return result
}

// firstNonEmpty returns the first of its values that is not an empty string
func firstNonEmpty(values ...string) string {

	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

// truncateRunes shortens a string to at most maxRunes runes
func truncateRunes(value string, maxRunes int) string {

	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}

	return string([]rune(value)[:maxRunes])
}
