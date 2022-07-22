package set

// testPerson implements Value[string] interface, and can be used to test set structures.
type testPerson struct {
	id    string
	name  string
	email string
}

func (p testPerson) ID() string {
	return p.id
}
