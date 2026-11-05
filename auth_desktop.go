package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	sessionCredentialTarget = "WoWLogsUploader/session"
	signInTimeout           = 5 * time.Minute
)

// DesktopAccount is what the UI shows for the current credential.
type DesktopAccount struct {
	SignedIn  bool   `json:"signedIn"`
	Source    string `json:"source"` // "session", "token" or ""
	TokenType string `json:"tokenType"`
	IsPremium bool   `json:"isPremium"`
	Reason    string `json:"reason,omitempty"`
	UserName  string `json:"userName,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// pkceChallenge is RFC 7636 S256.
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (a *App) sessionToken() string {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if !a.sessionLoaded {
		a.sessionLoaded = true
		token, err := credentialRead(sessionCredentialTarget)
		if err != nil {
			log.Printf("[Auth] Could not read the stored session: %v\n", err)
		}
		a.sessionValue = token
	}
	return a.sessionValue
}

func (a *App) setSessionToken(token string) error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	var err error
	if token == "" {
		err = credentialDelete(sessionCredentialTarget)
	} else {
		err = credentialWrite(sessionCredentialTarget, token)
	}
	if err != nil {
		return err
	}
	a.sessionLoaded = true
	a.sessionValue = token
	return nil
}

// awaitSignInCallback serves one loopback request and returns its code. The state check stops
// another local page from completing someone else's sign-in.
func awaitSignInCallback(ctx context.Context, listener net.Listener, state string) (string, error) {
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state || query.Get("code") == "" {
			http.Error(w, "Invalid sign-in response.", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, signInDonePage)
		select {
		case done <- result{code: query.Get("code")}:
		default:
		}
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case done <- result{err: err}:
			default:
			}
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	select {
	case res := <-done:
		return res.code, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

const signInDonePage = `<!doctype html><html><head><meta charset="utf-8"><title>WoW Logs Uploader</title></head>
<body style="font-family:system-ui,sans-serif;background:#0c0a09;color:#fafaf9;display:flex;align-items:center;justify-content:center;height:100vh;margin:0">
<div><h2>You're signed in</h2><p>You can close this tab and return to WoW Logs Uploader.</p></div></body></html>`

func (a *App) exchangeSignInCode(code, verifier string) (string, error) {
	hostname, _ := os.Hostname()
	payload, err := json.Marshal(map[string]string{
		"code":       code,
		"verifier":   verifier,
		"deviceId":   a.ensureDeviceID(),
		"deviceName": hostname,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, a.apiBaseURL+"/api/auth/desktop/exchange", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	setNativeUploaderAPIHeaders(req)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach the server: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", newAPIError(resp.StatusCode, resp.Status, body)
	}
	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Token == "" {
		return "", fmt.Errorf("the server did not return a session")
	}
	return parsed.Token, nil
}

// StartSignIn opens the website and waits for the user to approve the app.
func (a *App) StartSignIn() (*DesktopAccount, error) {
	verifier, err := randomURLToken(48)
	if err != nil {
		return nil, err
	}
	state, err := randomURLToken(24)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("could not start the sign-in listener: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), signInTimeout)
	defer cancel()
	a.signInMu.Lock()
	if a.signInCancel != nil {
		a.signInCancel()
	}
	a.signInCancel = cancel
	a.signInURL = fmt.Sprintf("%s/auth/desktop?%s", a.viewLogURL, url.Values{
		"port":      {fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)},
		"state":     {state},
		"challenge": {pkceChallenge(verifier)},
	}.Encode())
	signInURL := a.signInURL
	a.signInMu.Unlock()

	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, signInURL)
	}

	code, err := awaitSignInCallback(ctx, listener, state)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("sign-in timed out, please try again")
		}
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("sign-in cancelled")
		}
		return nil, err
	}
	token, err := a.exchangeSignInCode(code, verifier)
	if err != nil {
		return nil, err
	}
	if err := a.setSessionToken(token); err != nil {
		return nil, fmt.Errorf("could not store the session: %w", err)
	}
	return a.GetAccount()
}

// ReopenSignInPage opens the pending sign-in page again if the browser tab was closed.
func (a *App) ReopenSignInPage() {
	a.signInMu.Lock()
	signInURL := a.signInURL
	a.signInMu.Unlock()
	if signInURL != "" && a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, signInURL)
	}
}

// CancelSignIn stops a sign-in in progress.
func (a *App) CancelSignIn() {
	a.signInMu.Lock()
	defer a.signInMu.Unlock()
	if a.signInCancel != nil {
		a.signInCancel()
		a.signInCancel = nil
	}
}

// SignOut ends the session here and on the server. A pasted API token stays in place.
func (a *App) SignOut() error {
	token := a.sessionToken()
	if token != "" {
		req, err := http.NewRequest(http.MethodPost, a.apiBaseURL+"/api/auth/desktop/logout", nil)
		if err == nil {
			setNativeUploaderAPIHeaders(req)
			req.Header.Set("X-API-Token", token)
			client := &http.Client{Timeout: 10 * time.Second}
			if resp, err := client.Do(req); err != nil {
				log.Printf("[Auth] Server sign-out failed (clearing locally): %v\n", err)
			} else {
				resp.Body.Close()
			}
		}
	}
	return a.setSessionToken("")
}

// GetAccount validates the current credential with the server and returns premium status.
func (a *App) GetAccount() (*DesktopAccount, error) {
	token := a.authToken()
	if token == "" {
		return &DesktopAccount{}, nil
	}
	source := "token"
	if strings.TrimSpace(a.sessionToken()) != "" {
		source = "session"
	}

	req, err := http.NewRequest(http.MethodGet, a.apiBaseURL+"/api/auth/desktop/me", nil)
	if err != nil {
		return nil, err
	}
	a.setAPIHeaders(req)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the server: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		if source == "session" {
			_ = a.setSessionToken("")
		}
		return &DesktopAccount{Reason: "expired"}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIError(resp.StatusCode, resp.Status, body)
	}

	var parsed struct {
		TokenType string `json:"tokenType"`
		IsPremium bool   `json:"isPremium"`
		Reason    string `json:"reason"`
		User      *struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("could not read the account: %w", err)
	}
	account := &DesktopAccount{
		SignedIn:  true,
		Source:    source,
		TokenType: parsed.TokenType,
		IsPremium: parsed.IsPremium,
		Reason:    parsed.Reason,
	}
	if parsed.User != nil {
		account.UserName = parsed.User.Name
		account.AvatarURL = parsed.User.AvatarURL
	}
	return account, nil
}
