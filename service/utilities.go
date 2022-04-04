package service

import (
	"github.com/benpate/exp"
)

// notDeleted ensures that a criteria expression does not include soft-deleted items.
func notDeleted(criteria exp.Expression) exp.Expression {
	return criteria.And(exp.Equal("journal.deleteDate", 0))
}
