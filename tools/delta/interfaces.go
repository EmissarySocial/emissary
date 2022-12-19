package delta

import "go.mongodb.org/mongo-driver/bson/primitive"

type BoolGetterSetter interface {
	BoolGetter
	BoolSetter
}

type BoolGetter interface {
	GetBool(string) bool
}

type BoolSetter interface {
	SetBool(string, bool) bool
}

type IntGetterSetter interface {
	IntGetter
	IntSetter
}
type IntGetter interface {
	GetInt(string) int
}

type IntSetter interface {
	SetInt(string, int) bool
}

type Int64GetterSetter interface {
	Int64Getter
	Int64Setter
}

type Int64Getter interface {
	GetInt64(string) int64
}

type Int64Setter interface {
	SetInt64(string, int64) bool
}

type FloatGetterSetter interface {
	FloatGetter
	FloatSetter
}

type FloatGetter interface {
	GetFloat(string) float64
}
type FloatSetter interface {
	SetFloat(string, float64) bool
}

type ObjectIDGetterSetter interface {
	ObjectIDGetter
	ObjectIDSetter
}

type ObjectIDGetter interface {
	GetObjectID(string) primitive.ObjectID
}

type ObjectIDSetter interface {
	SetObjectID(string, primitive.ObjectID) bool
}

type StringGetterSetter interface {
	StringGetter
	StringSetter
}
type StringGetter interface {
	GetString(string) string
}

type StringSetter interface {
	SetString(string, string) bool
}

type ChildGetter interface {
	GetChild(string) (any, bool)
}
