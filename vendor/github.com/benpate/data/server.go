// Package data provides a data structure for defining simple database filters.  This
// is not able to represent every imaginable query criteria, but it does a good job of making
// common criteria simple to format and pass around in your application.
package data

import (
	"context"
)

// Server is an abstract representation of a database and its connection information.
type Server interface {
	Session(context.Context) (Session, error)
}
