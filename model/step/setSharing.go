package step

import (
	"slices"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetSharing is a Step that forces a Stream's sharing settings to one of the magic Groups
type SetSharing struct {
	Role  string
	Group string
}

// NewSetSharing returns a fully parsed SetSharing object
func NewSetSharing(stepInfo mapof.Any) (SetSharing, error) {

	const location = "step.NewSetSharing"

	// RULE: Role is required
	role := stepInfo.GetString("role")

	if role == "" {
		return SetSharing{}, derp.BadRequest(location, "Role is required")
	}

	// RULE: Group must name one of the magic Groups (matching model.MagicRole* values)
	group := stepInfo.GetString("group")

	if !slices.Contains([]string{"anonymous", "authenticated", "owner"}, group) {
		return SetSharing{}, derp.BadRequest(location, "Group must be 'anonymous', 'authenticated', or 'owner'", group)
	}

	return SetSharing{
		Role:  role,
		Group: group,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetSharing) Name() string {
	return "set-sharing"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetSharing) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetSharing) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetSharing) RequiredRoles() []string {
	return []string{step.Role}
}
