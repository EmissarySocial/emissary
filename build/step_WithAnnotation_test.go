package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestAnnotationTypeParam confirms that only the two Annotation types pass through from ?type=
func TestAnnotationTypeParam(t *testing.T) {
	require.Equal(t, model.AnnotationTypeActor, annotationTypeParam("ACTOR"))
	require.Equal(t, model.AnnotationTypeDocument, annotationTypeParam("DOCUMENT"))
	require.Equal(t, "", annotationTypeParam(""))
	require.Equal(t, "", annotationTypeParam("Person"))
	require.Equal(t, "", annotationTypeParam("actor"))
}
