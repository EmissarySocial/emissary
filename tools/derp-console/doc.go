// Package derpconsole prints errors to the console in a readable format.
//
// It implements derp.Reporter, so it plugs into the same reporting chain as the MongoDB
// reporter in tools/derp-mongo, and is the reporter a development server uses.
//
// Output is colorized and unrolls a wrapped derp error into its full chain -- location,
// message, and any attached values at each level -- because the whole point of derp's
// wrapping is lost if only the outermost message reaches the operator.
package derpconsole
