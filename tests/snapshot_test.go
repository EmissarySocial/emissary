package tests

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// result is what one scenario did to one postCase, as stored in the golden files
type result struct {
	Error   string            `json:"error,omitempty"`   // The error the step returned
	Root    string            `json:"root,omitempty"`    // The root cause, when it differs from Error
	Panic   string            `json:"panic,omitempty"`   // A recovered panic, which halts the request in production
	Changes map[string]string `json:"changes,omitempty"` // Every stored value the POST changed, as "type value"
	Types   map[string]string `json:"types,omitempty"`   // The Go type found at each posted path afterwards

	// OrderDependent marks a result that changes with Go's random map order.  Only the
	// parts that were identical on every run are kept; see pinScenario.
	OrderDependent bool `json:"orderDependent,omitempty"`
}

// run applies one scenario to a fresh object, and reports what changed
func (c postCase) run(values url.Values) result {
	return c.runWith(func(object any) error { return c.Apply(object, values) })
}

// runOrdered applies one scenario, visiting the step's maps in the orders given
func (c postCase) runOrdered(values url.Values, orders [][]string) result {
	return c.runWith(func(object any) error { return c.ApplyOrdered(object, values, orders) })
}

// runWith applies a step to a fresh object, and reports what changed
func (c postCase) runWith(apply func(object any) error) (output result) {

	object := c.NewObject()
	before := snapshot(object)

	// A panic is current behavior too, so record it rather than crash the suite
	defer func() {
		if recovered := recover(); recovered != nil {
			output.Panic = fmt.Sprint(recovered)
		}
	}()

	if err := apply(object); err != nil {
		output.Error = err.Error()
		if root := derp.RootCause(err); root != nil && root.Error() != output.Error {
			output.Root = root.Error()
		}
	}

	output.Changes = diff(before, snapshot(object))
	output.Types = c.types(object)

	return output
}

// types reads each posted path back through the schema and names the type it holds
func (c postCase) types(object any) map[string]string {

	result := map[string]string{}

	for _, field := range c.Fields {

		if field.Element == nil {
			continue
		}

		path := field.Path
		if field.SchemaPath != "" {
			path = field.SchemaPath
		}

		value, err := c.Schema.Get(object, path)

		if err != nil {
			result[path] = "unreadable"
			continue
		}

		result[path] = fmt.Sprintf("%T", value)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// snapshot flattens an object into "path" -> "type value" pairs, through BSON, which is
// how every one of these objects is stored
func snapshot(object any) map[string]string {

	result := map[string]string{}

	encoded, err := bson.Marshal(object)
	if err != nil {
		result[""] = "unencodable: " + err.Error()
		return result
	}

	var document bson.D
	if err := bson.Unmarshal(encoded, &document); err != nil {
		result[""] = "undecodable: " + err.Error()
		return result
	}

	flatten("", document, result)
	return result
}

// flatten walks a decoded BSON value, writing one entry per leaf.  Empty containers are
// leaves too, so that creating one is a visible change.
func flatten(prefix string, value any, result map[string]string) {

	switch typed := value.(type) {

	case bson.D:
		if len(typed) == 0 {
			result[prefix] = "document {}"
			return
		}
		for _, element := range typed {
			flatten(join(prefix, element.Key), element.Value, result)
		}

	case bson.A:
		if len(typed) == 0 {
			result[prefix] = "array []"
			return
		}
		for index, item := range typed {
			flatten(prefix+"["+strconv.Itoa(index)+"]", item, result)
		}

	case primitive.DateTime:
		result[prefix] = "datetime " + typed.Time().UTC().Format(time.RFC3339Nano)

	case primitive.ObjectID:
		result[prefix] = "objectId " + typed.Hex()

	case nil:
		result[prefix] = "null"

	case string:
		result[prefix] = "string " + abbreviate(typed)

	default:
		result[prefix] = fmt.Sprintf("%T %v", typed, typed)
	}
}

// abbreviate keeps a long string's length, hash, and opening, so that a golden file pins
// its exact contents without repeating them
func abbreviate(value string) string {

	if len(value) <= abbreviateOver {
		return value
	}

	sum := sha256.Sum256([]byte(value))
	opening := []rune(value)

	if len(opening) > abbreviateKeep {
		opening = opening[:abbreviateKeep]
	}

	return fmt.Sprintf("%q... (%d bytes, sha256 %x)", string(opening), len(value), sum[:6])
}

// Strings longer than abbreviateOver bytes are stored as their first abbreviateKeep runes
const (
	abbreviateOver = 80
	abbreviateKeep = 40
)

// join appends a key to a dotted path
func join(prefix string, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// diff returns every path whose value differs between two snapshots
func diff(before map[string]string, after map[string]string) map[string]string {

	result := map[string]string{}

	for path, value := range after {
		if before[path] != value {
			result[path] = value
		}
	}

	for path := range before {
		if _, exists := after[path]; !exists {
			result[path] = "removed"
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// sortedFields lists a case's fields as "path (widget)", which pins the form itself
func (c postCase) sortedFields() []string {

	result := make([]string, 0, len(c.Fields))

	for _, field := range c.Fields {
		description := field.Path + " (" + field.Widget
		if field.Element == nil {
			description += ", not in schema"
		} else {
			description += ", " + fmt.Sprintf("%T", field.Element)
		}
		result = append(result, description+")")
	}

	sort.Strings(result)
	return result
}
