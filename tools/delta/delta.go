package delta

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Delta simplifies getting and setting values in a nested object.
type Delta struct {
	object  any
	errors  error
	changed bool
}

// NewDelta returns a fully initialized Delta object
func NewDelta(object any) Delta {
	return Delta{
		object: object,
	}
}

// SetBool tracks changes to a bool value and collects errors
func (d *Delta) SetBool(path string, value bool) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(BoolGetterSetter); ok {

			if getter.GetBool(path) != value {
				getter.SetBool(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set bool", path, value))
}

// SetFloat tracks changes to a float value and collects errors
func (d *Delta) SetFloat(path string, value float64) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(FloatGetterSetter); ok {

			if getter.GetFloat(path) != value {
				getter.SetFloat(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set float", path, value))
}

// SetInt tracks changes to an int value and collects errors
func (d *Delta) SetInt(path string, value int) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(IntGetterSetter); ok {

			if getter.GetInt(path) != value {
				getter.SetInt(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set int", path, value))
}

// SetInt64 tracks changes to an int64 value and collects errors
func (d *Delta) SetInt64(path string, value int64) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(Int64GetterSetter); ok {

			if getter.GetInt64(path) != value {
				getter.SetInt64(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set int64", path, value))
}

// SetObjectID tracks changes to an ObjectID value and collects errors
func (d *Delta) SetObjectID(path string, value primitive.ObjectID) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(ObjectIDGetterSetter); ok {

			if getter.GetObjectID(path) != value {
				getter.SetObjectID(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set ObjectID", path, value))
}

// SetString tracks changes to a string value and collects errors
func (d *Delta) SetString(path string, value string) {

	if leaf, ok := d.getLeaf(d.object, list.Dot(path)); ok {

		if getter, ok := leaf.(StringGetterSetter); ok {

			if getter.GetString(path) != value {
				getter.SetString(path, value)
				d.changed = true
			}

			return
		}
	}

	d.errors = derp.Append(d.errors, derp.NewInternalError("delta.SetBool", "Unable to set string", path, value))
}

// HasChanged returns TRUE if any of the values have been changed
func (d *Delta) HasChanged() bool {
	return d.changed
}

// Error returns derp.MultiError containing all errors that have been collected
func (d *Delta) Error() error {
	return d.errors
}

// getLeaf traverses the object tree, returning a writable leaf object that contains the specified path
func (d *Delta) getLeaf(object any, path list.List) (any, bool) {

	head, tail := path.Split()

	if tail.IsEmpty() {
		return object, true
	}

	if childGetter, ok := object.(ChildGetter); ok {

		if child, ok := childGetter.GetChild(head); ok {
			return d.getLeaf(child, tail)
		}
	}

	return nil, false
}
