package handler

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	sessionStore  = make(map[string]string) // session_id -> email
	sessionMutex  = sync.Mutex{}
	userPageLimit int
)

func InitSession() {
	if limitStr := os.Getenv("USER_PAGE_LIMIT"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			userPageLimit = val
			return
		}
	}
	userPageLimit = 5
}

// GenerateSecureToken returns a pseudo-random session ID (64-char)
func GenerateSecureToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// SetSession sets a new session ID in memory and in the client's cookie
func SetSession(w http.ResponseWriter, email string) {
	token := GenerateSecureToken(64)

	sessionMutex.Lock()
	sessionStore[token] = email
	sessionMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,                 //JS can't access the cookie
		Secure:   false,                // true if you use HTTPS
		SameSite: http.SameSiteLaxMode, //A more relaxed form of cross-site request protection, cookie is sent with secure, top-level navigation
	})
}

// GetSessionEmail returns the email for a valid session cookie
func GetSessionEmail(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		return "", false
	}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	email, ok := sessionStore[cookie.Value]
	return email, ok
}

// ClearSession deletes session ID from memory and clears client cookie
func ClearSession(w http.ResponseWriter, r *http.Request) {
	//Get the session cookie, and delete the session from server-side memory map
	cookie, err := r.Cookie("session_id")
	if err == nil {
		sessionMutex.Lock()
		delete(sessionStore, cookie.Value)
		sessionMutex.Unlock()

		//Overwrites the client cookie with empty value and expiry -1, which deletes it from browser.
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1, //Delete the cookie
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func GetUserPageLimit() int {
	return userPageLimit
}

// RequireLogin is middleware to protect authenticated routes
func RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	//Prevents caching to avoid going back after logout.
	return func(w http.ResponseWriter, r *http.Request) {
		setNoCacheHeaders(w)

		_, ok := GetSessionEmail(r)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// Helps prevent back-button access to protected content after logout.
// Don’t store or cache this page in any way. Always fetch a fresh copy from the server.
func setNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache") //Tells older clients not to cache the response
	w.Header().Set("Expires", "0")       //Here 0 means already expired,Forces the browser to treat the response as expired immediately.
}
