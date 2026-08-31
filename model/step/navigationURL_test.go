package step

import (
	"bytes"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Navigation URL Templates
 *
 * "redirect-to" once compiled its "url" as an html/template,
 * whose contextual escaper turns "&" into "&amp;" -- corrupting
 * any multi-parameter target on its way into the Location
 * header. Every other Step argument is a text/template, and the
 * safety of these two comes from uri.IsSafeRedirectURL rather
 * than from escaping, so both belong on the same engine.
 ******************************************/

// multiParameterURL is a target that an HTML escaper would corrupt
const multiParameterURL = "/.checkout?productId=123&return=/@me"

func TestRedirectTo_URLIsNotHTMLEscaped(t *testing.T) {

	step, err := NewRedirectTo(mapof.Any{"url": multiParameterURL})
	require.NoError(t, err)

	var result bytes.Buffer
	require.NoError(t, step.URL.Execute(&result, nil))
	require.Equal(t, multiParameterURL, result.String())
	require.NotContains(t, result.String(), "&amp;")
}

func TestForwardTo_URLIsNotHTMLEscaped(t *testing.T) {

	step, err := NewForwardTo(mapof.Any{"url": multiParameterURL})
	require.NoError(t, err)

	var result bytes.Buffer
	require.NoError(t, step.URL.Execute(&result, nil))
	require.Equal(t, multiParameterURL, result.String())
}

// TestRedirectTo_Defaults pins the parsed defaults, which differ from forward-to's on
// purpose: "redirect-to" answers a view, while "forward-to" ends a submission.
func TestRedirectTo_Defaults(t *testing.T) {

	step, err := NewRedirectTo(mapof.Any{"url": "/somewhere"})
	require.NoError(t, err)
	require.Equal(t, "both", step.Method)
	require.Equal(t, 307, step.StatusCode)
}

// TestForwardTo_DefaultMethodIsPost pins a default that is load-bearing, not incidental.
// StepAsModal returns without Halt on a partial GET and StepEditModelObject returns nil,
// so a GET that opens a modal form runs on through to the forward-to that follows it --
// and only this default keeps that GET from navigating away before the modal renders.
func TestForwardTo_DefaultMethodIsPost(t *testing.T) {

	step, err := NewForwardTo(mapof.Any{"url": "/somewhere"})
	require.NoError(t, err)
	require.Equal(t, "post", step.Method)
}
