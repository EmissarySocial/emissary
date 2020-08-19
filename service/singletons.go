package service

import "github.com/benpate/ghost/model"

var singletonTemplateService *Template

var singletonRealtimeBroker *RealtimeBroker

var singletonTemplateWatcher chan *model.Template
