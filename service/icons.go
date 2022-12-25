package service

import (
	"io"
)

type Icons struct{}

func (service Icons) Get(name string) string {
	switch name {

	// App Actions and Behaviors
	case "add":
		return service.get("plus")
	case "book":
		return service.get("book")
	case "book-fill":
		return service.get("book-fill")
	case "cancel":
		return service.get("x-lg")
	case "check-circle":
		return service.get("check-circle")
	case "check-circle-fill":
		return service.get("check-circle-fill")
	case "chevron-left":
		return service.get("chevron-left")
	case "chevron-right":
		return service.get("chevron-right")
	case "delete":
		return service.get("trash")
	case "delete-fill":
		return service.get("trash-fill")
	case "edit":
		return service.get("pencil-square")
	case "edit-fill":
		return service.get("pencil-square-fill")
	case "email":
		return service.get("envelope")
	case "email-fill":
		return service.get("envelope-fill")
	case "file":
		return service.get("file-earmark")
	case "file-fill":
		return service.get("file-earmark-fill")
	case "flag":
		return service.get("flag")
	case "flag-fill":
		return service.get("flag-fill")
	case "folder":
		return service.get("folder")
	case "folder-fill":
		return service.get("folder-fill")
	case "grip-vertical":
		return service.get("grip-vertical")
	case "grip-horizontal":
		return service.get("grip-horizontal")
	case "home":
		return service.get("house")
	case "home-fill":
		return service.get("house-fill")
	case "info":
		return service.get("info-circle")
	case "info-fill":
		return service.get("info-circle-fill")
	case "link":
		return service.get("link-45deg")
	case "location":
		return service.get("geo-alt")
	case "location-fill":
		return service.get("geo-alt-fill")
	case "loading":
		return service.get("arrow-clockwise")
	case "login":
		return service.get("box-arrow-in-right")
	case "person":
		return service.get("person")
	case "person-fill":
		return service.get("person-fill")
	case "save":
		return service.get("check-lg")
	case "search":
		return service.get("search")
	case "settings":
		return service.get("gear")
	case "settings-fill":
		return service.get("gear-fill")
	case "server":
		return service.get("hdd-stack")
	case "server-fill":
		return service.get("hdd-stack-fill")
	case "share":
		return service.get("share")
	case "share-fill":
		return service.get("share-fill")
	case "star":
		return service.get("star")
	case "star-fill":
		return service.get("star-fill")
	case "reply":
		return service.get("reply")
	case "reply-fill":
		return service.get("reply-fill")
	case "unlink":
		return service.get("link-45deg")
	case "upload":
		return service.get("upload")
	case "user":
		return service.get("person-circle")
	case "user-fill":
		return service.get("person-circle-fill")
	case "users":
		return service.get("people")
	case "users-fill":
		return service.get("people-fill")

		// Services
	case "activity-pub":
		return service.get("cloud")
	case "facebook":
		return service.get("facebook")
	case "github":
		return service.get("github")
	case "google":
		return service.get("google")
	case "instagram":
		return service.get("instagram")
	case "twitter":
		return service.get("twitter")
	case "rss":
		return service.get("rss")
	case "rss-fill":
		return service.get("rss-fill")
	case "rss-cloud":
		return service.get("cloud-arrow-down")
	case "rss-cloud-fill":
		return service.get("cloud-arrow-down-fill")
	case "stripe":
		return service.get("credit-card")
	case "stripe-fill":
		return service.get("credit-card-fill")
	case "websub":
		return service.get("cloud-arrow-down")
	case "websub-fill":
		return service.get("cloud-arrow-down-fill")

	// Content Types
	case "article":
		return service.get("file-text")
	case "article-fill":
		return service.get("file-text-fill")
	case "block":
		return service.get("slash-circle")
	case "block-fill":
		return service.get("slash-circle-fill")
	case "collection":
		return service.get("view-stacked")
	case "forward":
		return service.get("forward")
	case "forward-fill":
		return service.get("forward-fill")
	case "inbox":
		return service.get("inbox")
	case "inbox-fill":
		return service.get("inbox-fill")
	case "markdown":
		return service.get("markdown")
	case "markdown-fill":
		return service.get("markdown-fill")
	case "message":
		return service.get("chat-left-text")
	case "message-fill":
		return service.get("chat-left-text-fill")
	case "outbox":
		return service.get("envelope")
	case "outbox-fill":
		return service.get("envelope-fill")
	case "picture":
		return service.get("image")
	case "picture-fill":
		return service.get("image-fill")
	case "pictures":
		return service.get("images")
	case "shopping-cart":
		return service.get("cart")
	case "shopping-cart-fill":
		return service.get("cart-fill")
	case "video":
		return service.get("camera-video")
	case "video-fill":
		return service.get("camera-video-fill")
	}

	return service.get(name)
}

func (service Icons) get(name string) string {
	return `<i class="bi bi-` + name + `"></i>`
}

func (service Icons) Write(name string, writer io.Writer) {
	writer.Write([]byte(service.Get(name)))
}
