package content

import (
	"strconv"
	"strings"

	"github.com/benpate/path"
)

type PathMaker struct {
	Counter int
	Parent  *PathMaker
}

func NewPathMaker() PathMaker {
	return PathMaker{
		Counter: -1,
	}
}

// getPath generates a path.Path representation of this linked list
func (pm *PathMaker) getPath() path.Path {

	counter := strconv.Itoa(pm.Counter)

	if pm.Parent == nil {
		return path.Path{counter}
	}

	p := pm.Parent.getPath()
	return append(p, counter)
}

// Path returns the string value of the current path.
func (pm *PathMaker) Path(delim string) string {
	return strings.Join(pm.getPath(), delim)
}

// NextPath increments the path and returns it as a string value.
func (pm *PathMaker) NextPath(delim string) string {
	pm.Counter = pm.Counter + 1
	return strings.Join(pm.getPath(), delim)
}

// SubTree makes a new node in the linked list.
func (pm *PathMaker) SubTree() PathMaker {
	result := NewPathMaker()
	result.Parent = pm
	return result
}

// ID returns the string value of the current ID
func (pm *PathMaker) ID() string {
	return strconv.Itoa(pm.Counter)
}

// NextID increments the current ID and returns it as a string value
func (pm *PathMaker) NextID() string {
	pm.Counter = pm.Counter + 1
	return strconv.Itoa(pm.Counter)
}

// Rewind undoes the incremented values, in case you need
// to re-generate a list of IDs (for example, tab labels and tab containers)
func (pm *PathMaker) Rewind(steps int) {
	pm.Counter = pm.Counter - steps
}
