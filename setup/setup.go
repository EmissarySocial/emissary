package setup

import (
	"embed"
	_ "embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/config"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/browser"
)

//go:embed all:*.html
var setupFiles embed.FS

func Setup(staticFiles fs.FS) {

	fmt.Println("Starting Emissary Config Tool.")

	configStorage := config.Load()

	factory := server.NewFactory(configStorage)
	setupTemplates := template.Must(
		template.New("").
			Funcs(render.FuncMap()).
			ParseFS(setupFiles, "*.html"))

	e := echo.New()

	// Global middleware
	e.Use(middleware.Recover())
	e.Use(mw.Localhost())

	// Routes
	e.GET("/", getPage(factory, setupTemplates, "index.html"))
	e.GET("/server", getPage(factory, setupTemplates, "server.html"))
	e.POST("/server", postServer(factory))
	e.GET("/domains", getPage(factory, setupTemplates, "domains.html"))
	e.GET("/domains/:domain", getDomain(factory))
	e.POST("/domains/:domain", postDomain(factory))
	e.DELETE("/domains/:domain", deleteDomain(factory))

	// Static Content
	var contentHandler = echo.WrapHandler(http.FileServer(http.FS(staticFiles)))
	var contentRewrite = middleware.Rewrite(map[string]string{"/static/*": "/_static/$1"})

	e.GET("/static/**", contentHandler, contentRewrite)

	// Prepare to open a browser window AFTER the server is ready
	go func() {
		time.Sleep(time.Second * 1)
		browser.OpenURL("http://localhost:8080/")
	}()

	// Start the HTTP server
	fmt.Println("Starting HTTP server...")
	if err := e.Start(":8080"); err != nil {
		derp.Report(derp.Wrap(err, "setup.Setup", "Error starting HTTP server"))
	}
}

// getPage returns the index page for the server
func getPage(factory *server.Factory, templates *template.Template, templateID string) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		useWrapper := (ctx.Request().Header.Get("HX-Request") != "true")

		renderer := NewRenderer(factory, factory.Config())

		header := ctx.Response().Header()
		header.Set("Content-Type", "text/html")
		header.Set("Cache-Control", "no-cache")

		if useWrapper {
			if err := templates.ExecuteTemplate(ctx.Response().Writer, "_header.html", renderer); err != nil {
				derp.Report(derp.Wrap(err, "setup.getIndex", "Error rendering index page"))
			}
		}

		if err := templates.ExecuteTemplate(ctx.Response().Writer, templateID, renderer); err != nil {
			derp.Report(derp.Wrap(err, "setup.getIndex", "Error rendering index page"))
		}

		if useWrapper {
			if err := templates.ExecuteTemplate(ctx.Response().Writer, "_footer.html", renderer); err != nil {
				derp.Report(derp.Wrap(err, "setup.getIndex", "Error rendering index page"))
			}
		}
		return nil
	}
}

func postServer(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		body := maps.Map{}
		c := factory.Config()
		s := config.Schema()

		// Try to get the FORM DATA ONLY
		if err := (&echo.DefaultBinder{}).BindBody(ctx, &body); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "setup.postServer", "Invalid Input (BAD FORMAT)."))
		}

		// Try to update the configuration with the form data
		if err := s.SetAll(c, body); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "setup.postServer", "Invalid Input."))
		}

		// Try to save the configuration to the persistent storage
		if err := factory.UpdateConfig(c); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "setup.postServer", "Internal error saving config.  Try again later."))
		}

		return render.WrapInlineSuccess(ctx, "Updated on "+time.Now().Format("3:04:05 PM"))
	}
}

// getNewDomain displays the form for creating/editing a domain.
func getDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		var header string

		domainID := ctx.Param("domain")

		if domainID == "new" {
			header = "Add a Domain"
		} else {
			header = "Edit a Domain"
		}

		domain, err := factory.DomainByID(domainID)

		if err != nil {
			return derp.Wrap(err, "handler.GetServerDomain", "Error loading configuration")
		}

		lib := factory.FormLibrary()

		f := form.Form{
			Kind:  "layout-vertical",
			Label: header,
			Children: []form.Form{{
				Kind:        "text",
				Path:        "label",
				Label:       "Label",
				Description: "Admin-friendly label for this domain",
			}, {
				Kind:        "text",
				Path:        "hostname",
				Label:       "Hostname",
				Description: "Complete domain name (but no https:// or trailing slashes)",
			}, {
				Kind:        "text",
				Path:        "connectString",
				Label:       "MongoDB Connection String",
				Description: "Should look like mongodb://host:port/database",
			}, {
				Kind:        "text",
				Path:        "databaseName",
				Label:       "MongoDB Database Name",
				Description: "Name of the database to use on the server",
			}},
		}

		s := config.DomainSchema()

		formHTML, err := f.HTML(&lib, &s, domain)

		if err != nil {
			return derp.Wrap(err, "handler.GetServerDomain", "Error generating form")
		}

		result := render.WrapModalForm(ctx.Response(), "/domains/"+domain.DomainID, formHTML)

		return ctx.HTML(200, result)
	}
}

// postDomain updates/creates a domain
func postDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		domainID := ctx.Param("domain")

		// Try to load the existing domain.  If it does not exist, then create a new one.
		domain, _ := factory.DomainByID(domainID)

		input := maps.Map{}

		if err := (&echo.DefaultBinder{}).BindBody(ctx, &input); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.PostServerDomain", "Error binding form input"))
		}

		s := config.DomainSchema()

		if err := s.SetAll(&domain, input); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.PostServerDomain", "Error setting config values"))
		}

		if err := factory.PutDomain(domain); err != nil {
			return render.WrapInlineError(ctx, derp.Wrap(err, "handler.PostServerDomain", "Error saving domain"))
		}

		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

// deleteDomain deletes a domain from the configuration
func deleteDomain(factory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		// Get the domain ID
		domainID := ctx.Param("domain")

		// Delete the domain
		if err := factory.DeleteDomain(domainID); err != nil {
			return derp.Wrap(err, "handler.DeleteServerDomain", "Error deleting domain")
		}

		// Close the modal and return OK
		render.RefreshPage(ctx)
		return ctx.NoContent(http.StatusOK)
	}
}

// getSigninToDomain signs you in to the requested domain as an administrator
func getSigninToDomain(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.GetSigninToDomain"

	return func(ctx echo.Context) error {

		// Get the domain config requested in the URL (by index)
		domain, err := fm.DomainByID(ctx.Param("domain"))

		if err != nil {
			return derp.Wrap(err, location, "Error loading configuration")
		}

		// Get the real factory for this domain
		factory, err := fm.ByDomainName(domain.Hostname)

		if err != nil {
			return derp.Wrap(err, location, "Error loading Domain")
		}

		// Create a fake "User" record for the system administrator and sign in
		s := factory.Steranko()

		administrator := model.NewUser()
		administrator.DisplayName = "System Administrator"
		administrator.IsOwner = true

		if err := s.CreateCertificate(ctx, &administrator); err != nil {
			return derp.Wrap(err, location, "Error creating certificate")
		}

		// Redirect to the admin page of this domain
		return ctx.Redirect(http.StatusTemporaryRedirect, "//"+domain.Hostname+"/startup")
	}
}
