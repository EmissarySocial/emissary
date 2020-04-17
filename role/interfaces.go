package role

// Object interface defines all of the methods that must be available to create/update/delete objects in the database
type Object interface {
	ID() string
	IsNew() bool
	SetCreated(int64, string)
	SetUpdated(int64, string)
	SetDeleted(int64, string)
}
