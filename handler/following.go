package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// GetFollowingTunnel is a hack to work around the restrictions from SameSite
// cookies.  If the user is coming from another site, their Authentication
// cookies won't be sent because we use SameSite=Strict.  But they WILL be
// sent from this redirect.  So, it's another hop, but it's still better for
// users.
func GetFollowingTunnel(context echo.Context) error {

	message := `<html>
<head>
<meta http-equiv="refresh" content="0;URL='/@me/settings/following-add?url={uri}'"/>
</head>
<body>
<p><a href="/@me/settings/following-add?url={uri}">Redirecting...</p>
</body>
</html>`

	forwardURL := context.QueryParam("uri")
	message = strings.ReplaceAll(message, "{uri}", forwardURL)

	return context.HTML(http.StatusOK, message)
}
