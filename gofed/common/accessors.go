package common

import (
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/***************************
 * Getter Interfaces
 ***************************/

type BoolGetter interface {
	GetBool(string) (bool, error)
}

type IntGetter interface {
	GetInt(string) (int, error)
}

type Int64Getter interface {
	GetInt64(string) (int64, error)
}

type ObjectIDGetter interface {
	GetObjectID(string) (primitive.ObjectID, error)
}

type StringGetter interface {
	GetString(string) (string, error)
}

/***************************
 * Setter Interfaces
 ***************************/

type BoolSetter interface {
	SetBool(string, bool) error
}

type IntSetter interface {
	SetInt(string, int) error
}

type Int64Setter interface {
	SetInt64(string, int64) error
}

type ObjectIDSetter interface {
	SetObjectID(string, primitive.ObjectID) error
}

type StringSetter interface {
	SetString(string, string) error
}

/***************************
 * Generic Getters
 ***************************/

func GetBool(object any, name string) (bool, error) {

	if getter, ok := object.(BoolGetter); ok {
		return getter.GetBool(name)
	}

	return false, derp.NewInternalError("activitypub.GetBool", "Object does not implement BoolGetter interface", name)
}

func GetInt(object any, name string) (int, error) {

	if getter, ok := object.(IntGetter); ok {
		return getter.GetInt(name)
	}

	return 0, derp.NewInternalError("activitypub.GetInt", "Object does not implement IntGetter interface", name)
}

func GetInt64(object any, name string) (int64, error) {

	if getter, ok := object.(Int64Getter); ok {
		return getter.GetInt64(name)
	}

	return 0, derp.NewInternalError("activitypub.GetInt64", "Object does not implement Int64Getter interface", name)
}

func GetObjectID(object any, name string) (primitive.ObjectID, error) {

	if getter, ok := object.(ObjectIDGetter); ok {
		return getter.GetObjectID(name)
	}

	return primitive.NilObjectID, derp.NewInternalError("activitypub.GetObjectID", "Object does not implement ObjectIDGetter interface", name)
}

func GetString(object any, name string) (string, error) {

	if getter, ok := object.(StringGetter); ok {
		return getter.GetString(name)
	}

	return "", derp.NewInternalError("activitypub.GetString", "Object does not implement StringGetter interface", name)
}

/***************************
 * Generic Setters
 ***************************/

func SetBool(object any, name string, value bool) error {

	if setter, ok := object.(BoolSetter); ok {
		return setter.SetBool(name, value)
	}

	return derp.NewInternalError("activitypub.SetBool", "Object does not implement BoolSetter interface", name)
}

func SetInt(object any, name string, value int) error {

	if setter, ok := object.(IntSetter); ok {
		return setter.SetInt(name, value)
	}

	return derp.NewInternalError("activitypub.SetInt", "Object does not implement IntSetter interface", name)
}

func SetInt64(object any, name string, value int64) error {

	if setter, ok := object.(Int64Setter); ok {
		return setter.SetInt64(name, value)
	}

	return derp.NewInternalError("activitypub.SetInt64", "Object does not implement Int64Setter interface", name)
}

func SetObjectID(object any, name string, value primitive.ObjectID) error {

	if setter, ok := object.(ObjectIDSetter); ok {
		return setter.SetObjectID(name, value)
	}

	return derp.NewInternalError("activitypub.SetObjectID", "Object does not implement ObjectIDSetter interface", name)
}

func SetString(object any, name string, value string) error {

	if setter, ok := object.(StringSetter); ok {
		return setter.SetString(name, value)
	}

	return derp.NewInternalError("activitypub.SetString", "Object does not implement StringSetter interface", name)
}
