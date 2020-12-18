package datastruct

import "sort"

// StringSlice is a sortable, comparable array
type StringSlice []string

func NewStringSlice() StringSlice {
	return StringSlice([]string{})
}

// IsIdentical returns TRUE if this StringSlice contains exactly the same items as the passed-in value.
func (s StringSlice) IsIdentical(other StringSlice) bool {

	if len(s) != len(other) {
		return false
	}

	for i := range s {
		if s[i] != other[i] {
			return false
		}
	}

	return true
}

// ContainsString returns TRUE if this StringSlice contains the provided value
func (s StringSlice) ContainsString(value string) bool {

	// Find the index where the value SHOULD be
	index := sort.SearchStrings(s, value)

	// Return TRUE if the value is actually there
	return (index < len(s)) && (s[index] == value)
}

// ContainsStringSlice returns TRUE if this StringSlice contains every string in the provided value
func (s StringSlice) ContainsStringSlice(value StringSlice) bool {

	for _, v := range value {
		if s.ContainsString(v) == false {
			return false
		}
	}

	return true
}

// Sort sorts this slice using the buil-in pkg/sort library
func (s StringSlice) Sort() {
	sort.Sort(s)
}

// RemoveDuplicates removes duplicates from a sorted StringSlice
func (s *StringSlice) RemoveDuplicates() {

	index := 0

	for {
		// Stop when we reach the last item
		if index >= len(*s)-1 {
			break
		}

		// If the current element is the same as the next element...
		if (*s)[index] == (*s)[index+1] {
			// remove it from the list
			*s = append((*s)[:index-1], (*s)[index+1:]...)
		}

		index = index + 1
	}
}

// Len implements the sort.Interface interface
func (s StringSlice) Len() int {
	return len(s)
}

// Swap implements the sort.Interface interface
func (s StringSlice) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StringSlice) Less(i int, j int) bool {
	return s[i] < s[j]
}
