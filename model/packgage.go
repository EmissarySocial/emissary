package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Package represents an external resource that has been loaded into this server.  It models a subset the NPM `package.json` file format.
type Package struct {
	PackageID    primitive.ObjectID
	Name         string              `json:"name,omitempty"`
	Version      string              `json:"version,omitempty"`
	Description  string              `json:"description,omitempty"`
	ReadMe       string              `json:"-"`
	Keywords     []string            `json:"keywords,omitempty"`
	Homepage     string              `json:"homepage,omitempty"`
	Bugs         PackageBugLocations `json:"bugs,omitempty"`
	License      string              `json:"license"`
	Author       PackagePerson       `json:"author"`
	Contributors []PackagePerson     `json:"contributors"`
	Repository   PackageRepository   `json:"repository"`

	journal.Journal
}

// PackageBugLocations lists out the places where bugs can be reported
type PackageBugLocations struct {
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// PackagePerson defines the information tracked about an author or contributor
type PackagePerson struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// PackageRepository defines the information tracked about the repository
type PackageRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// ID returns the unique identifier for this package (used internally only.)
func (p Package) ID() string {
	return p.PackageID.Hex()
}
