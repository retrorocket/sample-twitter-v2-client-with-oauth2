# sample-twitter-v2-client-with-oauth2

Twitter API v2をOAuth 2.0で使用するサンプル

* このサンプルではscopeを`"tweet.read", "tweet.write", "users.read"`に設定している。
* トークン取得後ツイートを投稿する。

## 使い方

1. Twitterアプリのcallback urlを`http://localhost:18199/oauth2`に設定する
1. redisをデフォルトの6379ポートで起動する
1. 環境変数`CLIENT_ID`, `CLIENT_SECRET`を設定する
1. `go run client.go`
1. ブラウザで http://localhost:18199/ を開く
