package nodeinfo

// NodeInfo represents the JSON structure returned by a NodeInfo API call.
// http://nodeinfo.diaspora.software/protocol.html
type NodeInfo struct {
	// The NodeInfo 2.0/2.1 schemas mark every field below as required and set
	// "additionalProperties": false, so none of them may carry `omitempty` -- an omitted
	// required field is a hard validation failure. Marshalling a zero-value NodeInfo emits
	// `null` for the slice and map fields, which fails just as hard; always build one
	// through NewNodeInfo.
	Version           string            `json:"version"`           // The schema version (2.0 or 2.1)
	Software          SoftwareInfo      `json:"software"`          // Metadata about server software in use.
	Protocols         []string          `json:"protocols"`         // The protocols supported on this server [activitypub, buddycloud, dfrn, diaspora, libertree, ostatus, pumpio, tent, xmpp]
	Services          ServicesInfo      `json:"services"`          // The third party sites this server can connect to via their application API.
	OpenRegistrations bool              `json:"openRegistrations"` // Whether this server allows open self-registration.
	Usage             UsageInfo         `json:"usage"`             // Usage statistics for this server.
	Metadata          map[string]string `json:"metadata"`          // Free form key value pairs for software specific values. Clients should not rely on any specific key present.
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
	// Name and Version are required by both schemas. Repository and Homepage exist only in
	// 2.1, and `software` sets "additionalProperties": false, so they must drop out of a 2.0
	// document entirely rather than appear as empty strings.
	Name       string `json:"name"`                 // The canonical name of this server software. Must match ^[a-z0-9-]+$
	Version    string `json:"version"`              // The version of this server software.
	Repository string `json:"repository,omitempty"` // The url of the source code repository of this server software. (2.1 only)
	Homepage   string `json:"homepage,omitempty"`   // The url of the homepage of this server software. (2.1 only)
}

// NewSoftwareInfo returns a fully initialized SoftwareInfo object
func NewSoftwareInfo() SoftwareInfo {
	return SoftwareInfo{}
}

// ServicesInfo represents the third party sites this server can connect to via their application API.
type ServicesInfo struct {
	// Both fields are required by `services`, and both are typed as arrays, so a nil slice
	// marshalled as `null` is invalid. NewServicesInfo initializes them to empty slices.
	Inbound  []string `json:"inbound"`  // The third party sites this server can connect to via their application API. [atom1.0 gnusocial imap pnut pop3 pumpio rss2.0 twitter]
	Outbound []string `json:"outbound"` // The third party sites this server can connect to via their application API. [atom1.0 blogger buddycloud diaspora dreamwidth drupal facebook friendica gnusocial google insanejournal libertree linkedin livejournal mediagoblin myspace pinterest pnut posterous pumpio redmatrix rss2.0 smtp tent tumblr twitter wordpress xmpp]
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
	// Users is required by `usage`. The counters are optional, and pointers so that a present
	// zero ("we counted, and the answer is none") stays distinct from an absent one ("we do not
	// disclose this") -- FEP-0151 requires the second rather than a fabricated zero.
	Users         UsersInfo `json:"users"`                   // Statistics about the users of this server.
	LocalPosts    *int64    `json:"localPosts,omitempty"`    // The amount of posts that were made by users that are registered on this server.
	LocalComments *int64    `json:"localComments,omitempty"` // The amount of comments that were made by users that are registered on this server.
}

// NewUsageInfo returns a fully initialized UsageInfo object
func NewUsageInfo() UsageInfo {
	return UsageInfo{
		Users: UsersInfo{},
	}
}

// UsersInfo represents statistics about the users of this server.
type UsersInfo struct {
	// Every field here is optional, and a pointer for the same reason as UsageInfo: a nil
	// counter drops out of the document entirely, which is how a server declines to disclose
	// a statistic. A non-nil zero is published as a real count of zero.
	Total          *int64 `json:"total,omitempty"`          // The total amount of on this server registered users.
	ActiveHalfyear *int64 `json:"activeHalfyear,omitempty"` // The amount of users that signed in at least once in the last 180 days.
	ActiveMonth    *int64 `json:"activeMonth,omitempty"`    // The amount of users that signed in at least once in the last 30 days.
}

// NewUsersInfo returns a fully initialized UsersInfo object
func NewUsersInfo() UsersInfo {
	return UsersInfo{}
}
