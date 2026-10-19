package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestAnnotationSchema returns the rosetta schema that describes a TestAnnotation
func TestAnnotationSchema(t *testing.T) {

	annotation := NewAnnotation()
	s := schema.New(AnnotationSchema())

	table := []tableTestItem{
		{"annotationId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"url", "http://example.com", nil},
		{"type", AnnotationTypeDocument, nil},
		{"name", "Test Annotation", nil},
		{"icon", "http://example.com/icon.png", nil},
		{"content", "This is a test annotation.", nil},
	}

	tableTest_Schema(t, &s, &annotation, table)
}

// TestAnnotationIsActor confirms that only the ACTOR type counts as an actor
func TestAnnotationIsActor(t *testing.T) {
	require.False(t, Annotation{}.IsActor())
	require.True(t, Annotation{Type: AnnotationTypeActor}.IsActor())
	require.False(t, Annotation{Type: AnnotationTypeDocument}.IsActor())
}

// TestAnnotationKind checks all four predicates on each type, including an unstamped Annotation
func TestAnnotationKind(t *testing.T) {

	actor := Annotation{Type: AnnotationTypeActor}
	require.True(t, actor.IsActor())
	require.False(t, actor.NotActor())
	require.False(t, actor.IsDocument())
	require.True(t, actor.NotDocument())

	document := Annotation{Type: AnnotationTypeDocument}
	require.False(t, document.IsActor())
	require.True(t, document.NotActor())
	require.True(t, document.IsDocument())
	require.False(t, document.NotDocument())

	// An Annotation that has not been stamped yet is neither
	unstamped := Annotation{}
	require.False(t, unstamped.IsActor())
	require.True(t, unstamped.NotActor())
	require.False(t, unstamped.IsDocument())
	require.True(t, unstamped.NotDocument())
}
