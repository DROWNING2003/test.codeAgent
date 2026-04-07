package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionFlow(t *testing.T) {
	s := newServer()
	handler := s.routes()

	loginBody, err := json.Marshal(map[string]string{"username": "alice"})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRec.Code, http.StatusOK)
	}

	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return cookie")
	}

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("session cookie missing or empty")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d", meRec.Code, http.StatusOK)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()
	handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRec.Code, http.StatusOK)
	}

	meAfterLogoutReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meAfterLogoutReq.AddCookie(sessionCookie)
	meAfterLogoutRec := httptest.NewRecorder()
	handler.ServeHTTP(meAfterLogoutRec, meAfterLogoutReq)
	if meAfterLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want %d", meAfterLogoutRec.Code, http.StatusUnauthorized)
	}
}

func TestLoginWithoutUsername(t *testing.T) {
	s := newServer()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()

	s.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
