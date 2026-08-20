package config

// Owner identifies the human being responsible for a Domain
type Owner struct {
	DisplayName    string `json:"displayName"     bson:"displayName"`
	Username       string `json:"username"        bson:"username"`
	EmailAddress   string `json:"emailAddress"    bson:"emailAddress"`
	PhoneNumber    string `json:"phoneNumber"     bson:"phoneNumber"`
	MailingAddress string `json:"mailingAddress"  bson:"mailingAddress"`
}

// NewOwner returns a fully initialized, empty Owner
func NewOwner() Owner {
	return Owner{}
}

// IsEmpty returns TRUE if this Owner has not been populated
func (owner Owner) IsEmpty() bool {
	return owner.Username == ""
}
