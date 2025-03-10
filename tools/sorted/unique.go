package sorted

// Unique returns a new slice that contains all of the UNIQUE VALUES
// from the input.  The input slice MUST BE SORTED or this algorithm
// will fail.
func Unique(slice []string) []string {

	result := make([]string, 0, len(slice))
	last := ""

	for _, entry := range slice {
		if entry != last {
			result = append(result, entry)
			last = entry
		}
	}

	return result
}
