package plugin

// PasswordHasher handles all encryption functions for passwords.
type PasswordHasher interface {

	// ID returns a string that uniquely identifies this plugin.
	ID() string

	// HashPassword returns a hashed value that can be (safely?) stored in a database
	HashPassword(plaintext string) (ciphertext string, error error)

	// CompareHashedValue checks that a plaintext value matches a stored ciphertext value.
	// OK returns TRUE if the values match.  Rehash returns TRUE if the hashing criteria has been updated
	// and a new hashed value should be stored in its place.
	CompareHashedPassword(plaintext string, ciphertext string) (OK bool, Rehash bool)
}
