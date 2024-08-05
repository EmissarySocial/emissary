package nodeinfo

// NodeInfo represents the JSON structure returned by a NodeInfo API call.
// http://nodeinfo.diaspora.software/protocol.html
type NodeInfo struct {
	Version           string            `json:"version"`                     // The schema version (2.1)
	Software          SoftwareInfo      `json:"software"`                    // Metadata about server software in use.
	Protocols         []string          `json:"protocols,omitempty"`         // The protocols supported on this server [activitypub, buddycloud, dfrn, diaspora, libertree, ostatus, pumpio, tent, xmpp]
	Services          ServicesInfo      `json:"services,omitempty"`          // The third party sites this server can connect to via their application API.
	OpenRegistrations bool              `json:"openRegistrations,omitempty"` // Whether this server allows open self-registration.
	Usage             UsageInfo         `json:"usage,omitempty"`             // Usage statistics for this server.
	Metadata          map[string]string `json:"metadata,omitempty"`          // Free form key value pairs for software specific values. Clients should not rely on any specific key present.
}

// NewNodeInfo returns a fully initialized NodeInfo object
func NewNodeInfo() NodeInfo {
	return NodeInfo{
		Protocols: make([]string, 0),
		Services:  NewServicesInfo(),
		Metadata:  make(map[string]string),
	}
}

// SoftwareInfo represents metadata about server software in use.
type SoftwareInfo struct {
	Name       string `json:"name"`       // The canonical name of this server software.
	Version    string `json:"version"`    // The version of this server software.
	Repository string `json:"repository"` // The url of the source code repository of this server software.
	Homepage   string `json:"homepage"`   // The url of the homepage of this server software.
}

// NewSoftwareInfo returns a fully initialized SoftwareInfo object
func NewSoftwareInfo() SoftwareInfo {
	return SoftwareInfo{}
}

// ServicesInfo represents the third party sites this server can connect to via their application API.
type ServicesInfo struct {
	Inbound  []string `json:"inbound,omitempty"`  // The third party sites this server can connect to via their application API. [atom1.0 gnusocial imap pnut pop3 pumpio rss2.0 twitter]
	Outbound []string `json:"outbound,omitempty"` // The third party sites this server can connect to via their application API. [atom1.0 blogger buddycloud diaspora dreamwidth drupal facebook friendica gnusocial google insanejournal libertree linkedin livejournal mediagoblin myspace pinterest pnut posterous pumpio redmatrix rss2.0 smtp tent tumblr twitter wordpress xmpp]
}

// NewServicesInfo returns a fully initialized ServicesInfo object
func NewServicesInfo() ServicesInfo {
	return ServicesInfo{
		Inbound:  make([]string, 0),
		Outbound: make([]string, 0),
	}
}

// UsageInfo represents usage statistics for this server.
type UsageInfo struct {
	Users         UsersInfo `json:"users,omitempty"`         // Statistics about the users of this server.
	LocalPosts    int       `json:"localPosts,omitempty"`    // The amount of posts that were made by users that are registered on this server.
	LocalComments int       `json:"localComments,omitempty"` // The amount of comments that were made by users that are registered on this server.
}

// NewUsageInfo returns a fully initialized UsageInfo object
func NewUsageInfo() UsageInfo {
	return UsageInfo{
		Users: UsersInfo{},
	}
}

// UsersInfo represents statistics about the users of this server.
type UsersInfo struct {
	Total          int `json:"total,omitempty"`          // The total amount of on this server registered users.
	ActiveHalfyear int `json:"activeHalfyear,omitempty"` // The amount of users that signed in at least once in the last 180 days.
	ActiveMonth    int `json:"activeMonth,omitempty"`    // The amount of users that signed in at least once in the last 30 days.
}

// NewUsersInfo returns a fully initialized UsersInfo object
func NewUsersInfo() UsersInfo {
	return UsersInfo{}
}
