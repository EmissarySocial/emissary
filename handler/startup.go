package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/domain"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Startup(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.Startup"

	return func(ctx echo.Context) error {

		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error finding domain")
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

		// Fall through..  we're done.  Jump to admin.
		return ctx.Redirect(http.StatusTemporaryRedirect, "/admin")
	}
}

// StartupUsers prompts users to create an initial admin account on this server
func StartupUsers(fm *server.Factory, factory *domain.Factory, ctx echo.Context) error {

	s := schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"displayname": schema.String{Format: "no-html"},
				"username":    schema.String{Format: "no-html"},
				"password":    schema.String{MinLength: null.NewInt(12)},
			},
		},
	}

	// IF POST, THEN TRY TO CREATE A NEW ADMIN ACCOUNT
	if ctx.Request().Method == http.MethodPost {

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

		return ctx.Redirect(http.StatusTemporaryRedirect, "/startup")
	}

	// OTHERWISE, DISPLAY THE USER FORM
	b := html.New()
	pageHeader(ctx, b, "Let's Get Started")

	b.Div().Class("align-center")
	b.H1().InnerHTML("Let's Set Up Your Whisperverse Server").Close()
	b.Div().Class("space-below")
	b.I("fa-8x fa-solid fa-volume-xmark gray20").Close()
	b.Close()
	b.Close()

	b.H2().InnerHTML("Step 1. Create an Administrator Account").Close()
	b.Div().Class("space-below").InnerHTML("Create an account for yourself that you'll use to sign in and manage your server.")

	b.Form(http.MethodPost, "/startup").EndBracket()

	library := fm.FormLibrary()
	f := form.Form{
		Kind: "layout-vertical",
		Children: []form.Form{{
			Kind:        "text",
			Path:        "displayname",
			Label:       "Your Name",
			Description: "Choose your publicly visible name.  You can always change it later.",
			Options: form.Map{
				"autocomplete": "OFF",
			},
		}, {
			Kind:        "text",
			Path:        "username",
			Label:       "Username",
			Description: "The name you'll use to sign in.",
			Options: form.Map{
				"autocomplete": "OFF",
			},
		}, {
			Kind:        "text",
			Path:        "password",
			Label:       "Password",
			Description: "At least 12 characters. Don't reuse passwords. Don't make it guessable.",
			Options: form.Map{
				"autocomplete": "OFF",
			},
		}},
	}
	formHTML, err := f.HTML(&library, &s, nil)

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

	s := schema.Schema{
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"blog":        schema.Boolean{},
				"photo-album": schema.Boolean{},
				"forum":       schema.Boolean{},
			},
		},
	}

	f := form.Form{
		Kind: "layout-vertical",
		Children: []form.Form{{
			Kind:        "checkbox",
			Path:        "blog",
			Label:       "Blog",
			Description: "Create and publish full-length blog articles.  Automatically organized by post date",
		}, {
			Kind:        "checkbox",
			Path:        "photo-album",
			Label:       "Photo Album",
			Description: "Upload photographs into an online album",
		}, {
			Kind:        "checkbox",
			Path:        "forum",
			Label:       "Discussion Forum",
			Description: "Realtime chat page for all your visitors, organized into discussion topics.",
		}},
	}

	library := fm.FormLibrary()
	if ctx.Request().Method == http.MethodPost {
	}

	b := html.New()

	formHTML, err := f.HTML(&library, &s, nil)

	if err != nil {
		return derp.Wrap(err, location, "Error generating form output")
	}

	b.WriteString(formHTML)

	return ctx.HTML(http.StatusOK, b.String())
}
