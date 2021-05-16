package transaction

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
)

type NilTransaction datatype.Map

func (txn NilTransaction) Execute(c *content.Content) error {
	return derp.New(500, "content.NilTransaction", "Unrecognized Transaction Type", txn)
}

func (txn NilTransaction) Description() string {
	return "Nil Transaction"
}
