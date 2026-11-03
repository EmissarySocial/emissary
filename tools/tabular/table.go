package tabular

// Table is the result of reading one data file
type Table struct {
	Format   Format    // Syntax the file was read as
	Rows     []Row     // Every row that was read without a problem, in the file's order
	Problems []Problem // Every row that could not be read, in the file's order
}

// HasProblems returns TRUE if any row could not be read
func (table Table) HasProblems() bool {
	return len(table.Problems) > 0
}
