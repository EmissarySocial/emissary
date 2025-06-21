package step

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

func requiredStates(steps ...Step) []string {

	result := make([]string, 0)

	for _, step := range steps {

		if required := step.RequiredStates(); len(required) > 0 {
			result = append(result, required...)
		}
	}

	return result
}

func requiredRoles(steps ...Step) []string {

	result := make([]string, 0)

	for _, step := range steps {

		if required := step.RequiredRoles(); len(required) > 0 {
			result = append(result, required...)
		}
	}

	return result
}
