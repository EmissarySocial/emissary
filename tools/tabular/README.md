# tabular

Reads a data file into rows, so that each row can become one record. It was written for collections, which turn each row into a child stream, but it knows nothing about streams.

## Formats

| Format | Shape |
| --- | --- |
| CSV, TSV | A header row of paths, then one row per record |
| JSON | A top-level array of objects |
| YAML | One document holding a list of mappings |

`Detect` picks the format from a specific Content-Type, then the file name or URL's extension, then the first meaningful line of the content. `text/plain` is treated as no hint, because many hosts send every file that way.

## Rows and paths

Every format produces the same `Table`: rows of `Cell`s, each pairing a dotted path with a value. A CSV heading is already a path. JSON and YAML objects flatten, so `{"location":{"latitude":1}}` and the heading `location.latitude` give the same cell. Keys keep the file's order.

CSV and TSV values are strings. JSON and YAML keep their own numbers, booleans, nulls, and arrays; an object inside an array becomes a `map[string]any`.

## Problems and errors

A bad row becomes a `Problem` naming its row, line, and column, and reading continues, so an author sees every bad row at once. Reading stops after `MaxProblems`, with one more problem saying so.

A bad file returns an error instead: one too large, not UTF-8, with broken syntax, with more rows or columns than the limits, or in the wrong shape, such as a JSON object wrapping the array.

## Limits

`Limits` caps the bytes, rows, columns, bytes per cell, nesting depth, and problems. A zero value takes the default from `DefaultLimits`. YAML aliases are refused outright, because expanding them can multiply a small file without limit.
