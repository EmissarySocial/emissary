package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Startup(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.Startup"

	return func(ctx echo.Context) error {

		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error finding domain")
		}

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewUnauthorizedError(location, "Unauthorized")
		}

		// Find/Create new database record for the domain.
		domainService := factory.Domain()
		domain := model.NewDomain()

		if err := domainService.Load(&domain); err != nil {
			if derp.NotFound(err) {
				if err := domainService.Save(&domain, "Created by startup wizard"); err != nil {
					return derp.Wrap(err, location, "Error creating domain record")
				}
			} else {
				return derp.Wrap(err, location, "Error searching for domain record")
			}
		}

		// If there are no groups, then add some defaults...
		groupService := factory.Group()
		groupCount, err := groupService.Count(ctx.Request().Context(), exp.All())

		if err != nil {
			return derp.Wrap(err, location, "Error counting groups")
		}

		if groupCount == 0 {
			for _, label := range []string{"Friends", "Editors", "Internet Randos"} {
				group := model.NewGroup()
				group.Label = label
				if err := groupService.Save(&group, "Created by Startup"); err != nil {
					return derp.Wrap(err, location, "Error creating group", label)
				}
			}
		}

		// If there are no users in the database, then display the USERS page
		userService := factory.User()
		userCount, err := userService.Count(ctx.Request().Context(), exp.All())

		if err != nil {
			return derp.Wrap(err, location, "Error counting users")
		}

		if userCount == 0 {
			return StartupUsers(fm, factory, ctx)
		}

		// If there are no streams in the database, then display the STREAMS page
		streamService := factory.Stream()
		streamCount, err := streamService.Count(ctx.Request().Context(), exp.Equal("parentId", primitive.NilObjectID))

		if err != nil {
			return derp.Wrap(err, location, "Error counting streams")
		}

		if streamCount == 0 {
			return StartupStreams(fm, factory, ctx)
		}

		// Fall through..  we're done.  Display "next steps" page
		return StartupDone(fm, ctx)
	}
}

// StartupUsers prompts users to create an initial admin account on this server
func StartupUsers(fm *server.Factory, factory *domain.Factory, ctx echo.Context) error {

	s := schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"displayname": schema.String{Format: "no-html"},
				"username":    schema.String{Format: "no-html"},
				"password":    schema.String{MinLength: 12},
			},
		},
	}

	// IF POST, THEN TRY TO CREATE A NEW ADMIN ACCOUNT
	if ctx.Request().Method == http.MethodPost {

		// Collect and validate the form information
		body := map[string]string{}

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, "handler.GetStartupUsername", "Error binding request body")
		}

		if err := s.Validate(body); err != nil {
			return derp.Wrap(err, "handler.GetStartupUsername", "Invalid form data")
		}

		// Create a new user record and save it to the database
		userService := factory.User()

		user := model.NewUser()
		user.SetPassword(body["password"])
		user.Username = body["username"]
		user.DisplayName = body["displayname"]
		user.IsOwner = true

		if err := userService.Save(&user, ""); err != nil {
			return derp.Wrap(err, "handler.GetStartupUsername", "Error saving user")
		}

		// Sign in as the new user
		s := factory.Steranko()
		if err := s.CreateCertificate(ctx, &user); err != nil {
			return derp.Wrap(err, "handler.GetStartupUsername", "Error signing in new user")
		}

		// Redirect to the next page (this forces a "GET" request)
		return ctx.Redirect(http.StatusSeeOther, "/startup?refresh=true")
	}

	icons := factory.Icons()

	// OTHERWISE, DISPLAY THE USER FORM
	b := html.New()
	pageHeader(ctx, b, "Let's Get Started")

	b.Div().Class("align-center")
	b.H1().InnerHTML("Let's Set Up Your Emissary Server").Close()
	b.Div().Class("space-below", "gray20")
	icons.Write("flag", b)
	b.Close()
	b.Close()

	b.Div().Class("pure-g")
	b.Div().Class("pure-u-md-1-6", "pure-u-lg-1-4").Close()
	b.Div().Class("pure-u-1", " pure-u-md-2-3", "pure-u-lg-1-2")

	b.H2().InnerHTML("Step 1/3 - Create an Administrator Account").Close()
	b.Div().Class("space-below").InnerHTML("Create an account for yourself that you'll use to sign in and manage your server.")

	b.Form(http.MethodPost, "/startup").EndBracket()

	userSetupForm := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{{
			Type:        "text",
			Path:        "displayname",
			Label:       "Your Name",
			Description: "Choose your publicly visible name.  You can always change it later.",
			Options: maps.Map{
				"autocomplete": "OFF",
			},
		}, {
			Type:        "text",
			Path:        "username",
			Label:       "Username",
			Description: "The name you'll use to sign in.",
			Options: maps.Map{
				"autocomplete": "OFF",
			},
		}, {
			Type:        "text",
			Path:        "password",
			Label:       "Password",
			Description: "At least 12 characters. Don't reuse passwords. Don't make it guessable.",
			Options: maps.Map{
				"autocomplete": "OFF",
			},
		}},
	}
	formHTML, err := form.Editor(s, userSetupForm, nil, factory.LookupProvider())

	if err != nil {
		return derp.Wrap(err, "handler.GetStartupUsername", "Error generating username form")
	}

	b.WriteString(formHTML)
	b.Button().Type("submit").Class("primary").InnerHTML("Create My Account &raquo;").Close()

	return ctx.HTML(http.StatusOK, b.String())
}

// StartupStreams prompts the administrator to choose the top-level
// items on this server.
func StartupStreams(fm *server.Factory, factory *domain.Factory, ctx echo.Context) error {

	const location = "handler.StartupStreams"

	streamService := factory.Stream()

	if ctx.Request().Method == http.MethodPost {

		body := maps.New()

		if err := ctx.Bind(&body); err != nil {
			return derp.Wrap(err, location, "Error binding request body")
		}

		streams := make([]model.Stream, 0)

		if isHome, ok := body.GetBool("home"); isHome && ok {
			stream := model.NewStream()
			stream.Document = model.DocumentLink{
				Label: "Welcome",
			}
			stream.TemplateID = "article-editorjs"
			stream.StateID = "published"
			stream.Token = "home"
			streams = append(streams, stream)
		}

		if isBlog, ok := body.GetBool("blog"); isBlog && ok {
			stream := model.NewStream()
			stream.Document = model.DocumentLink{
				Label: "Blog",
			}
			stream.TemplateID = "folder"
			stream.Token = "blog"
			stream.Data["format"] = "CARDS"
			stream.Data["showImages"] = "SHOW"
			streams = append(streams, stream)
		}

		if isAlbum, ok := body.GetBool("album"); isAlbum && ok {
			stream := model.NewStream()
			stream.Document = model.DocumentLink{
				Label: "Photo Album",
			}
			stream.TemplateID = "photo-album"
			stream.Token = "photos"
			streams = append(streams, stream)
		}

		// Try to add each new stream to the database.
		for index, stream := range streams {
			stream.Rank = index

			if err := streamService.Save(&stream, "Created by startup wizard"); err != nil {
				return derp.Wrap(err, location, "Error saving stream", stream)
			}
		}

		return ctx.Redirect(http.StatusSeeOther, "/startup")
	}

	b := html.New()
	pageHeader(ctx, b, "Let's Get Started")

	b.Div().Class("card")
	b.Div().Class("bold").InnerHTML("Step 2 of 3")
	b.H1().InnerHTML("How Do You Want To Use This Server?").Close()

	b.H3().InnerHTML("Choose which starter pages to put in your navigation bar.  You can always make changes later.").Close()

	b.Form(http.MethodPost, "/startup").EndBracket()

	defaultStreamsForm := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{{
			Type:        "toggle",
			Path:        "home",
			Options:     maps.Map{"true-text": "Home Page", "false-text": "Home Page"},
			Description: "Landing page when visitors first reach your site.",
		}, {
			Type:        "toggle",
			Path:        "blog",
			Options:     maps.Map{"true-text": "Blog Folder", "false-text": "Blog Folder"},
			Description: "Create and publish articles.  Automatically organized by date.",
		}, {
			Type:        "toggle",
			Path:        "album",
			Options:     maps.Map{"true-text": "Photo Album", "false-text": "Photo Album"},
			Description: "Upload and share photographs.",
		}},
	}

	s := schema.New(schema.Object{Properties: schema.ElementMap{
		"home":  schema.Boolean{},
		"blog":  schema.Boolean{},
		"album": schema.Boolean{},
		"forum": schema.Boolean{},
	}})

	formHTML, _ := form.Editor(s, defaultStreamsForm, nil, factory.LookupProvider())

	b.WriteString(formHTML)
	b.Button().Type("submit").Class("primary").InnerHTML("Set Up Initial Apps")

	return ctx.HTML(http.StatusOK, b.String())
}

// StartupDone prompts the administrator to take their next steps with the server
func StartupDone(factory *server.Factory, ctx echo.Context) error {

	icons := factory.Icons()

	b := html.New()

	pageHeader(ctx, b, "You're All Clear, Kid.")

	b.Div().Class("align-center")
	b.Div().Class("space-below", "text-gray", "text-2xl").EndBracket()
	icons.Write("check-circle", b)
	b.Close()
	b.H1().InnerHTML("Setup Is Complete").Close()
	b.H2().Class("gray70").InnerHTML("Here are some next steps you can take.").Close()
	b.Close()

	b.Table().Class("table", "space-above")

	b.TR().Role("link").Script("on click set window.location to '/home'")
	b.TD().Class("align-center", "text-lg").EndBracket()
	icons.Write("home", b)
	b.Close()
	b.TD().Style("width:100%")
	b.Div().Class("bold").InnerHTML("Visit Your New Home Page")
	b.Div().Class("gray70").InnerHTML("Start editing your new server.")
	b.Close()

	b.TR().Role("link").Script("on click set window.location to '/admin/users'")
	b.TD().Class("align-center", "text-lg").EndBracket()
	icons.Write("users", b)
	b.Close()
	b.TD().Style("width:100%")
	b.Div().Class("bold").InnerHTML("Invite People")
	b.Div().Class("gray70").InnerHTML("Send invitiations for other people to sign in and collaborate with you.")
	b.Close()

	return ctx.HTML(http.StatusOK, b.String())
}
