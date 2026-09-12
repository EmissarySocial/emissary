// Package currency renders money for display.
//
// Amounts move through Emissary as an integer number of minor units (cents, pence) paired
// with an ISO-4217 code, which is how payment processors report them and the only
// representation that does not lose precision.  UnitFormat turns that pair into a string a
// person can read, and Symbol maps a code to its display symbol.
//
// Zero-decimal currencies are handled here, so callers never have to know which codes divide
// by 100 and which do not.
package currency
