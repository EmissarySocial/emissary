package tabular

// FormatUnknown means that the format could not be determined
const FormatUnknown Format = ""

// FormatCSV is comma-separated values, with a header row of paths
const FormatCSV Format = "CSV"

// FormatTSV is tab-separated values, with a header row of paths
const FormatTSV Format = "TSV"

// FormatJSON is a JSON array of objects
const FormatJSON Format = "JSON"

// FormatYAML is a YAML sequence of mappings, in a single document
const FormatYAML Format = "YAML"

// DefaultMaxBytes is the largest file that Parse reads by default
const DefaultMaxBytes = 1 << 20

// DefaultMaxRows is the most rows that Parse reads by default
const DefaultMaxRows = 1024

// DefaultMaxColumns is the most cells in one row by default
const DefaultMaxColumns = 256

// DefaultMaxCellBytes is the longest text value in one cell by default
const DefaultMaxCellBytes = 64 << 10

// DefaultMaxDepth is the deepest nesting of objects and arrays within one row by default
const DefaultMaxDepth = 8

// DefaultMaxProblems is the most Problems that Parse reports by default before it stops
const DefaultMaxProblems = 100

// byteOrderMark is the UTF-8 byte-order mark that spreadsheet exports often put first
const byteOrderMark = "\uFEFF"
