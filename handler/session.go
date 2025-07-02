package handler

import (
	"os"
	"strconv"

	"github.com/gorilla/sessions"
)

var (
	store         *sessions.CookieStore
	userPageLimit int
)

func InitSession() {
	store = sessions.NewCookieStore([]byte("super-secret-session-key"))
	store.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   3600,
		Path:     "/",
	}

	if limitStr := os.Getenv("USER_PAGE_LIMIT"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			userPageLimit = val
		} else {
			userPageLimit = 5 //if error
		}
	} else {
		userPageLimit = 5 //default
	}
}

func GetStore() *sessions.CookieStore {
	return store
}
