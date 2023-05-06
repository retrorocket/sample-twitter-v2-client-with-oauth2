package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	redisStore "gopkg.in/boj/redistore.v1"
)

///////// echo向けの設定 //////////

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {

	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func TweetView(c echo.Context) error {
	return c.Render(http.StatusOK, "tweet.html", map[string]interface{}{})
}

func main() {
	router := NewRouter()
	// Start server
	router.Logger.Fatal(router.Start(":18199"))
}

func NewRouter() *echo.Echo {
	// Echo instance
	e := echo.New()
	store, err := redisStore.NewRediStore(10, "tcp", ":6379", "", []byte(securecookie.GenerateRandomKey(32)))
	if err != nil {
		panic(err)
	}
	e.Use(session.Middleware(store))

	// Setting template
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}
	e.Renderer = renderer

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", GetRedirectUrl)
	e.GET("/oauth2", GetToken)
	e.GET("/tweet", TweetView)
	e.POST("/createtweet", CreateTweet)
	e.GET("/refresh", Refresh)

	return e
}
