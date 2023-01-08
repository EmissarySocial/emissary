package gofed

// Federating implements the pub.FederatingProtocol, which is only needed if an
// application wants to do the S2S (Server-to-server, or federating) ActivityPub
// protocol. It supplements the pub.CommonBehavior interface with the additional
// methods required by a federating application.
type Federating struct {
	database Database
}

func NewFederatingProtocol(database Database) Federating {
	return Federating{
		database: database,
	}
}
