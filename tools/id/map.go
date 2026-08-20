package id

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Map is a set of named ObjectIDs that can be read and written through a rosetta schema
type Map map[string]primitive.ObjectID

// MapSchema returns the rosetta schema that describes a Map
func MapSchema() schema.Element {
	return schema.Object{
		Wildcard: schema.String{Format: "objectId"},
	}
}

// NewMap returns a fully initialized, empty Map
func NewMap() Map {
	return make(Map, 0)
}

/******************************************
 * Map Attributes
 ******************************************/

// IsZero returns TRUE if this Map contains no entries
func (m Map) IsZero() bool {
	return len(m) == 0
}

// NotZero returns TRUE if this Map contains at least one entry
func (m Map) NotZero() bool {
	return len(m) != 0
}

// Length returns the number of entries in this Map
func (m Map) Length() int {
	if m == nil {
		return 0
	}
	return len(m)
}

// Keys returns the names of every entry in this Map, in no particular order
func (m Map) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// IsMap returns TRUE, declaring this type a map for schema traversal. Implements schema.MapTyper.
func (m Map) IsMap() bool {
	return true
}

// Exists returns TRUE if this Map contains an entry with the provided key
func (m Map) Exists(key string) bool {
	_, exists := m[key]
	return exists
}

/******************************************
 * Schema Getter/Setter Interfaces
 ******************************************/

// GetStringOK retrieves the string representation of the ObjectID for the given key.
func (m Map) GetStringOK(key string) (string, bool) {
	if id, ok := m[key]; ok {
		return id.Hex(), true
	}
	return "", false
}

// GetString retrieves the string representation of the ObjectID for the given key.
func (m Map) GetString(key string) string {
	result, _ := m.GetStringOK(key)
	return result
}

// SetString sets the ObjectID for the given key (if the value is properly formatted)
func (m Map) SetString(key string, value string) bool {

	if value == "" {
		delete(m, key)
		return true
	}

	if id, err := primitive.ObjectIDFromHex(value); err == nil {

		if id.IsZero() {
			delete(m, key)
			return true
		}

		m[key] = id
		return true
	}

	return false
}

// SetDelta sets the value of the given key and returns TRUE if the value was changed
func (m Map) SetDelta(key string, value primitive.ObjectID) bool {

	// If the new value is the same as the old one, then return FALSE
	if value == m[key] {
		return false
	}

	// Otherwise set the new value and return TRUE
	m[key] = value
	return true
}
