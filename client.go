package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"context"
	"fmt"
	"log"

	twitter "github.com/g8rswimmer/go-twitter/v2"
)

///////// Twitter APIを使うための設定 //////////

// OAuth2.Configの生成
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

// code_verifier, code_challenge用のrandom byte生成
func randomBytesInHex(count int) (string, error) {
	buf := make([]byte, count)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", fmt.Errorf("Could not generate %d random bytes: %v", count, err)
	}

	return hex.EncodeToString(buf), nil
}

// Authorization Request用のURL生成
func GetRedirectUrl(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 10,
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

	return c.Redirect(http.StatusSeeOther, config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("response_type", "code"),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	))
}

// 認可コードを TokenURL に提示して token を取得する
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
	token, err := config.Exchange(
		oauth2.NoContext,
		code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return err
	}
	sess.Values["token"] = token.AccessToken
	sess.Values["refreshtoken"] = token.RefreshToken
	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/tweet")
}

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

// Twitter API v2 でツイートを投稿する
func CreateTweet(c echo.Context) error {
	sess, _ := session.Get("session", c)
	session_token, _ := sess.Values["token"].(string)

	text := c.FormValue("text")

	client := &twitter.Client{
		Authorizer: authorize{
			Token: session_token,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}

	req := twitter.CreateTweetRequest{
		Text: text,
	}

	tweetResponse, err := client.CreateTweet(context.Background(), req)
	if err != nil {
		log.Panicf("create tweet error: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, tweetResponse)
}

// RefreshTokenを使ってAccessTokenを再取得する
func Refresh(c echo.Context) error {
	sess, _ := session.Get("session", c)
	refreshtoken, _ := sess.Values["refreshtoken"].(string)
	token := new(oauth2.Token)
	token.AccessToken = ""
	token.RefreshToken = refreshtoken
	token.Expiry = time.Now()
	newtoken, err := config.TokenSource(c.Request().Context(), token).Token()
	if err != nil {
		return err
	}
	sess.Values["token"] = newtoken.AccessToken
	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusSeeOther, "/tweet")
}
