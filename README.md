# sample-twitter-v2-client-with-oauth2

Twitter API v2 を OAuth 2.0 で使用するサンプル

- トークン取得後ツイートを投稿する。
- このサンプルでは[POST /2/tweets](https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/post-tweets) と [Refresh Token](https://developer.twitter.com/en/docs/authentication/oauth-2-0/authorization-code)を使用するため、scope を`"tweet.read", "tweet.write", "users.read", "offline.access"`に設定している。

## 使い方

1. Twitter アプリの callback url を`http://localhost:18199/oauth2`に設定する
1. redis をデフォルトの 6379 ポートで起動する
1. 環境変数`CLIENT_ID`, `CLIENT_SECRET`を設定する
1. `go run .`
1. ブラウザで http://localhost:18199/ を開く

## 主な構成

- Web フレームワーク： echo v4
- Twitter API v2 クライアント： go-twitter
- セッションストア： Redis
