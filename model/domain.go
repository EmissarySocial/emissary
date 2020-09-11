package model

// Domain represents an account or node on this server.
type Domain struct {
	DomainID         string `json:"domainId" bson:"_id"`                         // This is the internal ID for the domain.  It should not be available via the web service.
	Name             string `json:"name"     bson:"name"`                        // Fully qualified domain name (without protocol)
	Forward          string `json:"forward,omitempty"  bson:"forward,omitempty"` // If present, then all requests for this domain should be forwarded to the designated new domain.
	ConnectionString string `json:"connectionString"`                            // MongoDB connection string
}
