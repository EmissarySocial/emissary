package build

import (
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// requireTestBuilder is a Builder whose only data is the value that a condition reads
type requireTestBuilder struct {
	Builder
	value string
}

// Value returns the value that the test conditions read
func (b requireTestBuilder) Value() string { return b.value }

// factory implements the Builder interface; no step under test reaches the factory
func (b requireTestBuilder) factory() Factory { return unusedFactory{} }

// requireTestPipeline parses a pipeline from its configuration, failing the test if it is invalid
func requireTestPipeline(t *testing.T, steps ...mapof.Any) []step.Step {
	t.Helper()

	parsed, err := step.NewPipeline(steps, nil)
	require.NoError(t, err)
	return parsed
}

// runRequire runs a parsed pipeline, which also proves the step's ExecutableStep case
func runRequire(steps []step.Step, value string, method ActionMethod) PipelineResult {
	return Pipeline(steps).Execute(nil, requireTestBuilder{value: value}, io.Discard, method)
}

// TestStepRequire_Met verifies that a met requirement lets the rest of the pipeline run, in both phases
func TestStepRequire_Met(t *testing.T) {

	steps := requireTestPipeline(t,
		mapof.Any{"do": "require", "condition": "{{eq .Value `yes`}}"},
		mapof.Any{"do": "add-event", "event": "ran", "method": "both"},
	)

	for _, method := range []ActionMethod{ActionMethodGet, ActionMethodPost} {
		result := runRequire(steps, "yes", method)
		require.NoError(t, result.Error, "method %v", method)
		require.False(t, result.Halt, "method %v", method)
		require.Equal(t, mapof.String{"ran": "true"}, result.Events, "method %v", method)
	}
}

// TestStepRequire_NotMet verifies that an unmet requirement halts with a 400 before any later step runs
func TestStepRequire_NotMet(t *testing.T) {

	steps := requireTestPipeline(t,
		mapof.Any{"do": "require", "condition": "{{eq .Value `yes`}}"},
		mapof.Any{"do": "add-event", "event": "ran", "method": "both"},
	)

	for _, method := range []ActionMethod{ActionMethodGet, ActionMethodPost} {
		result := runRequire(steps, "no", method)
		require.True(t, result.Halt, "method %v", method)
		require.Error(t, result.Error, "method %v", method)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), "method %v", method)
		require.Empty(t, result.Events, "method %v: no step after the requirement runs", method)
	}
}

// TestStepRequire_OnlyTrueIsTrue verifies that a condition passes only when it renders exactly "true"
func TestStepRequire_OnlyTrueIsTrue(t *testing.T) {

	steps := requireTestPipeline(t, mapof.Any{"do": "require", "condition": "{{.Value}}"})

	passes := func(value string) {
		t.Helper()
		result := runRequire(steps, value, ActionMethodPost)
		require.NoError(t, result.Error, "value %q", value)
		require.False(t, result.Halt, "value %q", value)
	}

	refuses := func(value string) {
		t.Helper()
		result := runRequire(steps, value, ActionMethodPost)
		require.True(t, result.Halt, "value %q", value)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), "value %q", value)
	}

	passes("true")
	refuses("false")
	refuses("")
	refuses("TRUE")
	refuses("True")
	refuses(" true")
	refuses("true\n")
	refuses("1")
	refuses("yes")
	refuses("\xff")
}

// TestStepRequire_AgreesWithIf verifies that a requirement passes exactly when an "if" step on
// the same condition takes its "then" branch
func TestStepRequire_AgreesWithIf(t *testing.T) {

	requireSteps := requireTestPipeline(t, mapof.Any{"do": "require", "condition": "{{.Value}}"})
	ifSteps := requireTestPipeline(t, mapof.Any{
		"do":        "if",
		"condition": "{{.Value}}",
		"then":      []mapof.Any{{"do": "add-event", "event": "then", "method": "both"}},
	})

	for _, value := range []string{"true", "false", "", "TRUE", " true", "1", "0", "yes"} {
		passed := !runRequire(requireSteps, value, ActionMethodPost).Halt
		tookThen := runRequire(ifSteps, value, ActionMethodPost).Events["then"] == "true"
		require.Equal(t, tookThen, passed, "value %q", value)
	}
}

// TestStepRequire_ConditionError verifies that a condition that cannot render halts as a server
// error, not as an unmet requirement
func TestStepRequire_ConditionError(t *testing.T) {

	steps := requireTestPipeline(t, mapof.Any{"do": "require", "condition": "{{.Missing}}"})

	for _, method := range []ActionMethod{ActionMethodGet, ActionMethodPost} {
		result := runRequire(steps, "true", method)
		require.True(t, result.Halt, "method %v", method)
		require.Error(t, result.Error, "method %v", method)
		require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(result.Error), "method %v", method)
	}
}

// TestStepRequire_SharedAcrossRequests verifies that concurrent requests through one parsed step
// each get their own answer
func TestStepRequire_SharedAcrossRequests(t *testing.T) {

	steps := requireTestPipeline(t, mapof.Any{"do": "require", "condition": "{{eq .Value `yes`}}"})

	values := []string{"yes", "no", "yes", "no", "yes", "no", "yes", "no"}
	halted := make([]bool, len(values))

	var wait sync.WaitGroup
	for index, value := range values {
		wait.Go(func() {
			halted[index] = runRequire(steps, value, ActionMethodPost).Halt
		})
	}
	wait.Wait()

	for index, value := range values {
		require.Equal(t, value != "yes", halted[index], "request %d (%q)", index, value)
	}
}

// FuzzStepRequire_Value verifies that any rendered value either passes or refuses with a 400,
// never panics, and passes only when it is exactly "true"
func FuzzStepRequire_Value(f *testing.F) {

	f.Add("true")
	f.Add("false")
	f.Add("")
	f.Add(" true")
	f.Add("\xff\x00")

	parsed, err := step.NewPipeline([]mapof.Any{{"do": "require", "condition": "{{.Value}}"}}, nil)
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, value string) {

		result := runRequire(parsed, value, ActionMethodPost)

		if value == "true" {
			require.NoError(t, result.Error)
			require.False(t, result.Halt)
			return
		}

		require.True(t, result.Halt, "value %q", value)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), "value %q", value)
	})
}
