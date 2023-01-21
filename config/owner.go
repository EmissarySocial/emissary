package config

type Owner struct {
	DisplayName    string `json:"displayName"     bson:"displayName"`
	Username       string `json:"username"        bson:"username"`
	EmailAddress   string `json:"emailAddress"    bson:"emailAddress"`
	PhoneNumber    string `json:"phoneNumber"     bson:"phoneNumber"`
	MailingAddress string `json:"mailingAddress"  bson:"mailingAddress"`
}

func NewOwner() Owner {
	return Owner{}
}
