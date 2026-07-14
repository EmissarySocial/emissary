package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// pushWorkerJS is the Web Push service worker.  It MUST be served from the site root so its scope
// covers the whole origin (a worker under /resources/ could only control that sub-path).  It stays
// intentionally dumb: the server renders the notification payload (title/body/icon/url).
const pushWorkerJS = `// Emissary Web Push service worker
self.addEventListener("push", (event) => {
	let data = {};
	try { data = event.data ? event.data.json() : {}; } catch (e) { data = {}; }
	const title = data.title || "New notification";
	event.waitUntil(self.registration.showNotification(title, {
		body: data.body || "",
		icon: data.icon || undefined,
		data: { url: data.url || "/" }
	}));
});

self.addEventListener("notificationclick", (event) => {
	event.notification.close();
	const url = (event.notification.data && event.notification.data.url) || "/";
	event.waitUntil(clients.openWindow(url));
});
`

// GetWebPushWorker serves the Web Push service worker JavaScript at the site root.
func GetWebPushWorker(ctx echo.Context) error {
	ctx.Response().Header().Set("Content-Type", "text/javascript; charset=utf-8")
	ctx.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return ctx.String(http.StatusOK, pushWorkerJS)
}
