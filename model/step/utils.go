package step

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// parseMethod reads a Step's "method" property, which names the request methods that the
// Step runs for.
func parseMethod(stepInfo mapof.Any, defaultValue string) (string, error) {

	result := strings.ToLower(first(stepInfo.GetString("method"), defaultValue))

	// RULE: An unrecognized value is an error, not a silent default. Steps guard on this
	// value with hand-written comparisons, and the two shapes of that guard disagree about
	// what an unknown value means: an allow-list ("method == post") runs the Step for
	// NOTHING, while a deny-list ("method != post") runs it for EVERYTHING. Rejecting the
	// typo at Template load time is what keeps those two shapes interchangeable.
	switch result {

	case "get", "post", "both":
		return result, nil
	}

	return "", derp.Validation("Step 'method' must be one of: get, post, both", result)
}

// first is a cheapy little function to pick the first "non-zero" value from
// a list of values.
func first[T comparable](values ...T) T {

	var zero T

	for _, value := range values {
		if value != zero {
			return value
		}
	}

	return zero
}

// requiredStates returns every state named by the provided steps, in order
func requiredStates(steps ...Step) []string {

	result := make([]string, 0)

	for _, step := range steps {

		if required := step.RequiredStates(); len(required) > 0 {
			result = append(result, required...)
		}
	}

	return result
}

// requiredRoles returns every role named by the provided steps, in order
func requiredRoles(steps ...Step) []string {

	result := make([]string, 0)

	for _, step := range steps {

		if required := step.RequiredRoles(); len(required) > 0 {
			result = append(result, required...)
		}
	}

	return result
}
