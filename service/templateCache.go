package service

import "github.com/benpate/ghost/model"

// templateCache is a package-level variable that stores all loaded templates in memory.
// TODO: this is just a placeholder for now.  We should use a real in-memory templating system long before release.
var templateCache map[string]*model.Template

func init() {
	templateCache = map[string]*model.Template{}
}
