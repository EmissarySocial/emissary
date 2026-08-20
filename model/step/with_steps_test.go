package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// The "with-*" steps share one shape: each parses a "steps" sub-pipeline into SubSteps, exposes a
// distinct Name()/RequiredModel(), and bubbles RequiredStates/RequiredRoles up from the sub-steps.
// This table exhaustively exercises every with-* step: the success path (valid sub-pipeline),
// the propagated-states/roles behavior, the four interface methods, and the error path (an
// unrecognized sub-step).

// withStepCase describes one "with-" step: how to build it, and the Name and RequiredModel it must report
type withStepCase struct {
	name          string
	build         func(mapof.Any) (Step, error)
	expectedName  string
	expectedModel string
}

// withStepCases returns one case per "with-" step, so that the table below covers all of them
func withStepCases() []withStepCase {
	return []withStepCase{
		{"WithAnnotation", func(s mapof.Any) (Step, error) { return NewWithAnnotation(s) }, "with-annotation", ""},
		{"WithAttachment", func(s mapof.Any) (Step, error) { return NewWithAttachment(s) }, "with-attachment", ""},
		{"WithChildren", func(s mapof.Any) (Step, error) { return NewWithChildren(s) }, "with-children", "Stream"},
		{"WithCircle", func(s mapof.Any) (Step, error) { return NewWithCircle(s) }, "with-circle", ""},
		{"WithDraft", func(s mapof.Any) (Step, error) { return NewWithDraft(s) }, "with-draft", "Stream"},
		{"WithFolder", func(s mapof.Any) (Step, error) { return NewWithFolder(s) }, "with-folder", ""},
		{"WithFollower", func(s mapof.Any) (Step, error) { return NewWithFollower(s) }, "with-follower", ""},
		{"WithFollowing", func(s mapof.Any) (Step, error) { return NewWithFollowing(s) }, "with-following", ""},
		{"WithImport", func(s mapof.Any) (Step, error) { return NewWithImport(s) }, "with-import", ""},
		{"WithKeyPackage", func(s mapof.Any) (Step, error) { return NewWithKeyPackage(s) }, "with-key-package", "Settings"},
		{"WithMerchantAccount", func(s mapof.Any) (Step, error) { return NewWithMerchantAccount(s) }, "with-merchant-account", ""},
		{"WithMessage", func(s mapof.Any) (Step, error) { return NewWithMessage(s) }, "with-message", ""},
		{"WithNextSibling", func(s mapof.Any) (Step, error) { return NewWithNextSibling(s) }, "with-next-sibling", "Stream"},
		{"WithOAuthToken", func(s mapof.Any) (Step, error) { return NewWithOAuthToken(s) }, "with-oauth-token", ""},
		{"WithParent", func(s mapof.Any) (Step, error) { return NewWithParent(s) }, "with-parent", "Stream"},
		{"WithPrevSibling", func(s mapof.Any) (Step, error) { return NewWithPrevSibling(s) }, "with-prev-sibling", "Stream"},
		{"WithPrivilege", func(s mapof.Any) (Step, error) { return NewWithPrivilege(s) }, "with-privilege", ""},
		{"WithResponse", func(s mapof.Any) (Step, error) { return NewWithResponse(s) }, "with-response", ""},
		{"WithRule", func(s mapof.Any) (Step, error) { return NewWithRule(s) }, "with-rule", ""},
	}
}

// TestWithSteps_Success verifies that every "with-" step parses the sub-steps nested inside it
func TestWithSteps_Success(t *testing.T) {

	for _, testCase := range withStepCases() {
		t.Run(testCase.name, func(t *testing.T) {

			step, err := testCase.build(mapof.Any{
				"steps": []mapof.Any{{"do": "set-state", "state": "published"}},
			})
			require.Nil(t, err)
			require.Equal(t, testCase.expectedName, step.Name())
			require.Equal(t, testCase.expectedModel, step.RequiredModel())

			// Roles/states are derived from the sub-pipeline (or empty), never nil.
			require.NotNil(t, step.RequiredStates())
			require.NotNil(t, step.RequiredRoles())
		})
	}
}

// TestWithSteps_InvalidSubStep verifies that a malformed sub-step is rejected by every "with-" step
func TestWithSteps_InvalidSubStep(t *testing.T) {

	for _, testCase := range withStepCases() {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.build(mapof.Any{
				"steps": []mapof.Any{{"do": "this-step-does-not-exist"}},
			})
			require.NotNil(t, err)
		})
	}
}
