// Package templatemap holds a set of named, pre-compiled Go templates.
//
// A Map is a plain map from name to *template.Template, which lets a configuration value be
// declared once as template source, compiled once at load time, and then executed many times
// without re-parsing on every use.
//
// Because the zero value of the map's element is a nil template, callers should look a name
// up before executing it rather than assuming every configured key produced a template.
package templatemap
