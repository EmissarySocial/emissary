package model

// Domain represents an account or node on this server.
type Domain struct {
	DomainID  string `json:"domainId"            bson:"_id"`               // This is the internal ID for the domain.  It should not be available via the web service.
	Label     string `json:"label"               bson:"label"`             // Human-friendly name displayed at the top of this domain
	BannerURL string `json:"bannerUrl,omitempty" bson:"bannerUrl"`         // URL of a banner image to display at the top of this domain
	Forward   string `json:"forward,omitempty"   bson:"forward,omitempty"` // If present, then all requests for this domain should be forwarded to the designated new domain.
}
