package model

// DomainSummary is an abbreviated Domain, used when a Domain is embedded in another document
type DomainSummary struct {
	Host     string
	Name     string
	IconURL  string
	ImageURL string
}
