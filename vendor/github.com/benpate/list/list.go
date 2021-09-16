package list

import "strings"

// Head returns the FIRST item in a string-based-list
func Head(value string, delimiter string) string {

	index := strings.Index(value, delimiter)

	if index == -1 {
		return value
	}

	return value[:index]
}

// Tail returns any values in the string-based-list AFTER the first item
func Tail(value string, delimiter string) string {
	index := strings.Index(value, delimiter)

	if index == -1 {
		return ""
	}

	return value[index+1:]
}

// RemoveLast returns the full list, with the last element removed.
func RemoveLast(value string, delimiter string) string {

	index := strings.LastIndex(value, delimiter)

	if index == -1 {
		return ""
	}

	return value[:index]
}

// Last returns the LAST item in a string-based-list
func Last(value string, delimiter string) string {

	index := strings.LastIndex(value, delimiter)

	if index == -1 {
		return value
	}

	return value[index+1:]
}

// Split returns the FIRST element, and the REST element in one function call
func Split(value string, delimiter string) (string, string) {

	index := strings.Index(value, delimiter)

	if index == -1 {
		return value, ""
	}

	return value[:index], value[index+1:]

}

// SplitTail behaves like Split, but with the TAIL instead of the HEAD.  It returns the REST element and the LAST element in one function call.
func SplitTail(value string, delimiter string) (string, string) {

	index := strings.LastIndex(value, delimiter)

	if index == -1 {
		return value, ""
	}

	return value[:index], value[index+1:]

}

// At returns the list vaue at a particular index
func At(value string, delimiter string, index int) string {

	if index == 0 {
		return Head(value, delimiter)
	}

	tail := Tail(value, delimiter)

	if tail == "" {
		return ""
	}

	return At(tail, delimiter, index-1)
}

// PushHead adds a new item to the beginning of the list
func PushHead(value string, newValue string, delimiter string) string {

	if len(value) == 0 {
		return newValue
	}

	return newValue + delimiter + value
}

// PushTail adds a new item to the end of the list
func PushTail(value string, newValue string, delimiter string) string {

	if len(value) == 0 {
		return newValue
	}

	return value + delimiter + newValue
}
