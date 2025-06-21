package step

import (
	"strconv"

	"github.com/benpate/rosetta/mapof"
)

// CacheURL is an action that can add new model objects of any type
type CacheURL struct {
	CacheControl string
}

// NewCacheURL returns a fully initialized CacheURL record
func NewCacheURL(stepInfo mapof.Any) (CacheURL, error) {

	cacheControl := ""

	// Switch between public and private caches
	if stepInfo.GetBool("private") {
		cacheControl = "private"
	} else {
		cacheControl = "public"
	}

	// Calculate the max-age
	maxAge := first(stepInfo.GetInt("max-age"), 3600)
	cacheControl += ", max-age=" + strconv.Itoa(maxAge)

	return CacheURL{
		CacheControl: cacheControl,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step CacheURL) Name() string {
	return "cache-url"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step CacheURL) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step CacheURL) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step CacheURL) RequiredRoles() []string {
	return []string{}
}
