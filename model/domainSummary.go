package model

// DomainSummary is an abbreviated Domain, used when a Domain is embedded in another document
type DomainSummary struct {
	Host     string `json:"host"`
	Name     string `json:"name"`
	IconURL  string `json:"iconUrl"`
	ImageURL string `json:"imageUrl"`
}
