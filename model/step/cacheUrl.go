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

// AmStep is here only to verify that this struct is a build pipeline step
func (step CacheURL) AmStep() {}
