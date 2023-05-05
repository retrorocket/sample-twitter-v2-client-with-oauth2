package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	redisStore "gopkg.in/boj/redistore.v1"

	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	twitter "github.com/g8rswimmer/go-twitter/v2"
)

var (
	config = oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  "http://localhost:18199/oauth2",
		Scopes: []string{
			"tweet.read", "tweet.write", "users.read", "offline.access",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
	}
)

func randomBytesInHex(count int) (string, error) {
	buf := make([]byte, count)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", fmt.Errorf("Could not generate %d random bytes: %v", count, err)
	}

	return hex.EncodeToString(buf), nil
}

// create Callback URL
func GetRedirectUrl(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 5,
		HttpOnly: true,
		Secure:   true,
	}
	state, stateErr := randomBytesInHex(24)
	if stateErr != nil {
		return stateErr
	}

	codeVerifier, verifierErr := randomBytesInHex(32) // 64 character string here
	if verifierErr != nil {
		return verifierErr
	}

	sha2 := sha256.New()
	io.WriteString(sha2, codeVerifier)
	codeChallenge := base64.RawURLEncoding.EncodeToString(sha2.Sum(nil))

	sess.Values["state"] = state
	sess.Values["verifier"] = codeVerifier
	err := sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, config.AuthCodeURL(state, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("response_type", "code"),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	))
}

func GetToken(c echo.Context) error {
	stateParam := c.QueryParam("state")
	sess, _ := session.Get("session", c)
	state, _ := sess.Values["state"]
	if state != stateParam {
		return c.NoContent(http.StatusForbidden)
	}
	code := c.QueryParam("code")
	if code == "" {
		return c.NoContent(http.StatusForbidden)
	}
	verifier, _ := sess.Values["verifier"].(string)
	token, err := config.Exchange(oauth2.NoContext, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return err
	}
	sess.Values["token"] = token.AccessToken
	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/tweet")
}

func TweetView(c echo.Context) error {
	return c.Render(http.StatusOK, "tweet.html", map[string]interface{}{})
}

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

func CreateTweet(c echo.Context) error {
	sess, _ := session.Get("session", c)
	session_token, _ := sess.Values["token"].(string)

	token := flag.String("token", session_token, "twitter API token")
	text := flag.String("text", c.FormValue("text"), "twitter text")
	flag.Parse()

	client := &twitter.Client{
		Authorizer: authorize{
			Token: *token,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}

	req := twitter.CreateTweetRequest{
		Text: *text,
	}
	fmt.Println("Callout to create tweet callout")

	tweetResponse, err := client.CreateTweet(context.Background(), req)
	if err != nil {
		log.Panicf("create tweet error: %v", err)
		return err
	}

	enc, err := json.MarshalIndent(tweetResponse, "", "    ")
	if err != nil {
		log.Panic(err)
		return err
	}
	return c.JSON(http.StatusOK, string(enc))
}

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

	return e
}
