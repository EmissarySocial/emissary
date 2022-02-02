package config

type SMTPConnection struct {
	Hostname string `path:"hostname" json:"hostname"` // Server name to connect to
	Username string `path:"username" json:"username"` // Username for authentication
	Password string `path:"password" json:"password"` // Password/secret for authentication
	TLS      bool   `path:"tls"      json:"tls"`      // If TRUE, then use TLS to connect
}
