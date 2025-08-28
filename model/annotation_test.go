package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestAnnotationSchema(t *testing.T) {

	annotation := NewAnnotation()
	s := schema.New(AnnotationSchema())

	table := []tableTestItem{
		{"annotationId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"url", "http://example.com", nil},
		{"name", "Test Annotation", nil},
		{"icon", "http://example.com/icon.png", nil},
		{"content", "This is a test annotation.", nil},
	}

	tableTest_Schema(t, &s, &annotation, table)
}
