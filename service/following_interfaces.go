package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
)

type followingImportFunc func(following *model.Following, transaction *http.Response, body *bytes.Buffer) error
