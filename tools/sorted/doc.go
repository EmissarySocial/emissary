// Package sorted provides set operations over slices that are already in sorted order.
//
// Because the inputs are known to be sorted, these functions walk each slice once instead of
// scanning repeatedly, so Contains, ContainsAll, and Unique cost linear rather than
// quadratic time on the sizes Emissary actually passes them.
//
// The precondition is not checked.  Passing an unsorted slice produces a wrong answer rather
// than an error, so sort first, or use the equivalents in rosetta/slice instead.
package sorted
