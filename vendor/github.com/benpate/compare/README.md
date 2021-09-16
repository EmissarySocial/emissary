# Compare

This library provides a number of helper functions that compare values of different or unknown data types.  It works around some of Go's more annoying issues with generics, and will probably be simplified or eliminated once the final generic proposals land in a stable release.

## Interface(value1, value2) int

Pass any two values into this function and it will attempt to convert them into comparable data types.

If value1 < value2, it returns -1
If value1 == value2, it returns 0
If value1 > value2, it returns 1

## IntXX Functions

This is a series of functions `Int()` `Int8()` `Int16()` `Int32()` `Int64()` that compare similar values of their corresponding types.

## FloatXX Functions

This is a series of functions `Float32()` `Float64()` that compare similar values of their corresponding types.

## String Functions 

There are several string comparison functions as well.  

`String()` compares two string values with the same signatures as above

`BeginsWith()` returns true if a string begins with a certain value
`Contains()` returns true if a string contains a certain value
`EndsWith()` returns true if a string ends with a certain value


## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making this library better, send in a pull request.  We're all in this together! 📚
