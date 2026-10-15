// Package tabular reads data files into rows, so that each row can become one record.
//
// It reads CSV, TSV, JSON, and YAML. Detect picks the format from a Content-Type, a
// file name, or the content itself, and Parse turns the bytes into a Table: an ordered
// list of Rows, each an ordered list of Cells that pair a dotted path with a value.
//
// CSV and TSV headings are already paths. JSON and YAML files are an array of objects,
// and nested objects flatten into dotted paths, so every format produces the same rows.
//
// Parse enforces Limits while it reads. A problem in one row becomes a Problem naming
// its line and column, so an author sees every bad row at once. A problem with the
// whole file, such as broken syntax or too many rows, is returned as an error instead.
package tabular
