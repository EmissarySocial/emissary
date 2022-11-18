package activitypub

import (
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/***************************
 * Getter Interfaces
 ***************************/

type GetBooler interface {
	GetBool(string) (bool, error)
}

type GetInter interface {
	GetInt(string) (int, error)
}

type GetInt64er interface {
	GetInt64(string) (int64, error)
}

type GetObjectIDer interface {
	GetObjectID(string) (primitive.ObjectID, error)
}

type GetStringer interface {
	GetString(string) (string, error)
}

/***************************
 * Setter Interfaces
 ***************************/

type SetBooler interface {
	SetBool(string, bool) error
}

type SetInter interface {
	SetInt(string, int) error
}

type SetInt64er interface {
	SetInt64(string, int64) error
}

type SetObjectIDer interface {
	SetObjectID(string, primitive.ObjectID) error
}

type SetStringer interface {
	SetString(string, string) error
}

/***************************
 * Generic Getters
 ***************************/

func GetBool(object any, name string) (bool, error) {

	if getter, ok := object.(GetBooler); ok {
		return getter.GetBool(name)
	}

	return false, derp.NewInternalError("activitypub.GetBool", "Object does not implement GetBooler interface", name)
}

func GetInt(object any, name string) (int, error) {

	if getter, ok := object.(GetInter); ok {
		return getter.GetInt(name)
	}

	return 0, derp.NewInternalError("activitypub.GetInt", "Object does not implement GetInter interface", name)
}

func GetInt64(object any, name string) (int64, error) {

	if getter, ok := object.(GetInt64er); ok {
		return getter.GetInt64(name)
	}

	return 0, derp.NewInternalError("activitypub.GetInt64", "Object does not implement GetInt64er interface", name)
}

func GetObjectID(object any, name string) (primitive.ObjectID, error) {

	if getter, ok := object.(GetObjectIDer); ok {
		return getter.GetObjectID(name)
	}

	return primitive.NilObjectID, derp.NewInternalError("activitypub.GetObjectID", "Object does not implement GetObjectIDer interface", name)
}

func GetString(object any, name string) (string, error) {

	if getter, ok := object.(GetStringer); ok {
		return getter.GetString(name)
	}

	return "", derp.NewInternalError("activitypub.GetString", "Object does not implement GetStringer interface", name)
}

/***************************
 * Generic Setters
 ***************************/

func SetBool(object any, name string, value bool) error {

	if setter, ok := object.(SetBooler); ok {
		return setter.SetBool(name, value)
	}

	return derp.NewInternalError("activitypub.SetBool", "Object does not implement SetBooler interface", name)
}

func SetInt(object any, name string, value int) error {

	if setter, ok := object.(SetInter); ok {
		return setter.SetInt(name, value)
	}

	return derp.NewInternalError("activitypub.SetInt", "Object does not implement SetInter interface", name)
}

func SetInt64(object any, name string, value int64) error {

	if setter, ok := object.(SetInt64er); ok {
		return setter.SetInt64(name, value)
	}

	return derp.NewInternalError("activitypub.SetInt64", "Object does not implement SetInt64er interface", name)
}

func SetObjectID(object any, name string, value primitive.ObjectID) error {

	if setter, ok := object.(SetObjectIDer); ok {
		return setter.SetObjectID(name, value)
	}

	return derp.NewInternalError("activitypub.SetObjectID", "Object does not implement SetObjectIDer interface", name)
}

func SetString(object any, name string, value string) error {

	if setter, ok := object.(SetStringer); ok {
		return setter.SetString(name, value)
	}

	return derp.NewInternalError("activitypub.SetString", "Object does not implement SetStringer interface", name)
}
