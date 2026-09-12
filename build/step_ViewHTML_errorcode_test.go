package build

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// unauthorizedSubject stands in for a Builder whose method fails a permission check,
// the way build.Stream.Parent does when NewStream refuses an anonymous visitor.
type unauthorizedSubject struct{}

func (unauthorizedSubject) Parent() (string, error) {
	return "", derp.Unauthorized("build.NewStream", "Unauthorized: Anonymous user is not authorized to perform this action")
}

// TestViewHTMLPreservesPermissionStatusCode guards the chain that turned a 401 into a 500.
//
// RULE: html/template reports a failing method as its own ExecError, so the derp error that
// StepViewHTML wraps is reached through TWO foreign layers.  The status code has to survive
// both, or a permission failure is reported (and logged) as a server fault.
func TestViewHTMLPreservesPermissionStatusCode(t *testing.T) {

	compiled := template.Must(template.New("view").Parse(`{{.Parent}}`))

	var buffer bytes.Buffer
	executeError := compiled.Execute(&buffer, unauthorizedSubject{})

	require.Error(t, executeError)
	require.Equal(t, 401, derp.ErrorCode(executeError), "the template layer must not erase the status code")

	// This is the exact wrap that StepViewHTML.execute applies to the template error.
	wrapped := derp.Wrap(executeError, "build.StepViewHTML.Get", "Executing template")
	require.Equal(t, 401, derp.ErrorCode(wrapped))

	// And the two frames the handler adds above it.
	wrapped = derp.Wrap(wrapped, "handler.getStreamPipeline", "Building page")
	wrapped = derp.Wrap(wrapped, "server.errorHandler", "Generating web page")

	require.Equal(t, 401, derp.ErrorCode(wrapped), "the reported status code must still be 401")
	require.True(t, derp.IsUnauthorized(wrapped), "errorHandler routes on this, so it must be true")
	require.Equal(t, "build.NewStream", derp.RootLocation(wrapped))
}
