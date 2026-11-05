package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// apiError carries the gateway's {message, error, code} body so users see the real reason
// (maintenance, quota, unsupported archive) instead of a bare status line.
type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string {
	return e.Message
}

func newAPIError(status int, statusText string, body []byte) *apiError {
	var parsed struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Code    string `json:"code"`
	}
	_ = json.Unmarshal(body, &parsed)

	message := strings.TrimSpace(parsed.Message)
	detail := strings.TrimSpace(parsed.Error)
	switch {
	case message != "" && detail != "":
		message = fmt.Sprintf("%s %s", message, detail)
	case message == "":
		message = detail
	}
	if message == "" {
		message = fmt.Sprintf("Server returned %s", statusText)
	}
	return &apiError{Status: status, Code: parsed.Code, Message: message}
}

// authToken is the credential sent as X-API-Token: the signed-in session first, then a token
// pasted under Settings → Advanced.
func (a *App) authToken() string {
	if token := strings.TrimSpace(a.sessionToken()); token != "" {
		return token
	}
	return strings.TrimSpace(a.config.ApiToken)
}

// setAPIHeaders marks the request as coming from the desktop app and adds the credential.
func (a *App) setAPIHeaders(req *http.Request) {
	setNativeUploaderAPIHeaders(req)
	if token := a.authToken(); token != "" {
		req.Header.Set("X-API-Token", token)
	}
}
