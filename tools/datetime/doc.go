// Package datetime carries a moment in time that forms can edit one piece at a time.
//
// DateTime wraps a time.Time and exposes its date, time, timezone, and Unix value as
// separate named fields, so an HTML form can bind a date picker and a time picker to the
// same underlying value without either one clobbering the other.
//
// The type satisfies rosetta's getter and setter interfaces and provides its own Schema, so
// it drops into a model object like any other schema-aware value, and it marshals to BSON as
// a single native date rather than as a nested document.
package datetime
