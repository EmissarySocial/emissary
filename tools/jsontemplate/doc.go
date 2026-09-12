// Package jsontemplate renders a Go template whose output is then parsed as JSON.
//
// It exists so a configuration value can be computed per request -- a webhook body, an API
// payload -- while still reaching the caller as structured data rather than as a string that
// every call site has to unmarshal for itself.
//
// By default the output is parsed as HJSON, which tolerates comments, unquoted keys, and
// trailing commas, so a hand-written template is forgiving.  WithStrictMode switches to the
// standard JSON parser for callers that want the stricter contract.
package jsontemplate
