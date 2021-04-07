package model

import "github.com/benpate/convert"

// A generic record for reading aribrary records from the database into a map[string]interface{}
type Generic map[string]interface{}

func NewGeneric() Generic {
	return make(Generic)
}

// ID returns the primary key of the object
func (g Generic) ID() string {
	return convert.String(g["_id"])
}

// IsNew returns TRUE if the object has not yet been saved to the database
func (g Generic) IsNew() bool {
	return true
}

// SetCreated stamps the CreateDate and UpdateDate of the object, and makes a note
func (g Generic) SetCreated(comment string) {

}

// SetUpdated stamps the UpdateDate of the object, and makes a note
func (g Generic) SetUpdated(comment string) {

}

// SetDeleted marks the object virtually "deleted", and makes a note
func (g Generic) SetDeleted(comment string) {

}
