package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

const sessionCookieName = "session_id"

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]string),
	}
}

func (s *sessionStore) create(username string) (string, error) {
	id, err := newSessionID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[id] = username
	s.mu.Unlock()

	return id, nil
}

func (s *sessionStore) getUser(sessionID string) (string, bool) {
	s.mu.RLock()
	username, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	return username, ok
}

func (s *sessionStore) delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

type server struct {
	store *sessionStore
}

func newServer() *server {
	return &server{
		store: newSessionStore(),
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/me", s.handleMe)
	mux.HandleFunc("/logout", s.handleLogout)
	return mux
}

type loginRequest struct {
	Username string `json:"username"`
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	sessionID, err := s.store.create(req.Username)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "login success"})
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.currentUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"username": username})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		s.store.delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "logout success"})
}

func (s *server) currentUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return s.store.getUser(cookie.Value)
}

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func main() {
	s := newServer()
	log.Println("server started on :8080")
	if err := http.ListenAndServe(":8080", s.routes()); err != nil {
		log.Fatal(err)
	}
}
