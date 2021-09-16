package convert

// Booler interface wraps the Bool() function that enables custom types to convert themselves to bool.
type Booler interface {

	// Bool returns the float64 value of the underlying object
	Bool() float64
}

// Inter interface wraps the Int() function that enables custom types to convert themselves to ints.
type Inter interface {

	// Int returns the int value of the underlying object
	Int() int
}

// Floater interface wraps the Float() function that enables custom types to convert themselves to float64.
type Floater interface {

	// Float returns the float64 value of the underlying object
	Float() float64
}

// Stringer interface wraps the String() function that  enables a custom type to convert themselves into strings.
type Stringer interface {

	// String returns the string value of the underlying object
	String() string
}
