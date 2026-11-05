package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPkceChallengeMatchesRFC7636Example(t *testing.T) {
	// RFC 7636 appendix B.
	got := pkceChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	if got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("got %s", got)
	}
}

func listenLoopback(t *testing.T) (net.Listener, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener, fmt.Sprintf("http://%s", listener.Addr().String())
}

func TestAwaitSignInCallbackRejectsWrongStateThenAcceptsCode(t *testing.T) {
	listener, base := listenLoopback(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		code, err := awaitSignInCallback(ctx, listener, "good-state")
		done <- result{code, err}
	}()

	bad, err := http.Get(base + "/callback?state=evil&code=stolen")
	if err != nil {
		t.Fatal(err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong state returned %d", bad.StatusCode)
	}

	good, err := http.Get(base + "/callback?state=good-state&code=abc")
	if err != nil {
		t.Fatal(err)
	}
	good.Body.Close()
	if good.StatusCode != http.StatusOK {
		t.Fatalf("good callback returned %d", good.StatusCode)
	}

	res := <-done
	if res.err != nil || res.code != "abc" {
		t.Fatalf("code=%q err=%v", res.code, res.err)
	}
}

func TestAwaitSignInCallbackStopsOnCancel(t *testing.T) {
	listener, _ := listenLoopback(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := awaitSignInCallback(ctx, listener, "s"); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestExchangeSignInCode(t *testing.T) {
	var sent map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/desktop/exchange" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&sent)
		_, _ = w.Write([]byte(`{"token":"wlds_abc"}`))
	}))
	defer server.Close()

	app := &App{apiBaseURL: server.URL, config: Config{DeviceID: "dev-1"}}
	token, err := app.exchangeSignInCode("code", "verifier")
	if err != nil {
		t.Fatal(err)
	}
	if token != "wlds_abc" || sent["code"] != "code" || sent["verifier"] != "verifier" || sent["deviceId"] != "dev-1" {
		t.Fatalf("token=%q sent=%v", token, sent)
	}
}

func TestExchangeSignInCodeShowsServerMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Sign-in expired. Please try again."}`))
	}))
	defer server.Close()
	app := &App{apiBaseURL: server.URL, config: Config{DeviceID: "dev-1"}}
	if _, err := app.exchangeSignInCode("c", "v"); err == nil || err.Error() != "Sign-in expired. Please try again." {
		t.Fatalf("got %v", err)
	}
}

func TestAuthTokenPrefersSessionOverPastedToken(t *testing.T) {
	app := &App{config: Config{ApiToken: " pasted "}}
	app.sessionLoaded = true
	if got := app.authToken(); got != "pasted" {
		t.Fatalf("got %q", got)
	}
	app.sessionValue = "wlds_session"
	if got := app.authToken(); got != "wlds_session" {
		t.Fatalf("got %q", got)
	}
}

func TestGetAccountWithPastedGuildToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Token") != "GUILD_x" {
			t.Errorf("token %q", r.Header.Get("X-API-Token"))
		}
		_, _ = w.Write([]byte(`{"tokenType":"GUILD","isPremium":true,"user":null}`))
	}))
	defer server.Close()
	app := &App{apiBaseURL: server.URL, config: Config{ApiToken: "GUILD_x"}}
	app.sessionLoaded = true
	account, err := app.GetAccount()
	if err != nil {
		t.Fatal(err)
	}
	if !account.SignedIn || !account.IsPremium || account.Source != "token" || account.TokenType != "GUILD" {
		t.Fatalf("account: %+v", account)
	}
}
