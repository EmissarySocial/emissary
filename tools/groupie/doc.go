// Package groupie tracks the most recent value in a sequence.
//
// Templates use it to insert a group header only when a repeating value changes -- a date
// heading above the first item of each day, for example -- without having to look ahead or
// keep their own state.
//
// A Groupie is stateful and single-pass: Header reports TRUE the first time it sees a value
// and every time that value differs from the one before it.  Create one per rendered list.
package groupie
