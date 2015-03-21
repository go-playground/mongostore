package store

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// TokenGetSeter is the interface for setting and receiving the cookie token
type TokenGetSeter interface {
	GetToken(r *http.Request, name string) (string, error)
	SetToken(w http.ResponseWriter, name, value string, options *sessions.Options)
}

// CookieToken struct
type CookieToken struct{}

// GetToken returns a Cookie valie and eny errors
func (c *CookieToken) GetToken(r *http.Request, name string) (string, error) {

	cookie, err := r.Cookie(name)

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// SetToken sets the sessions cookie
func (c *CookieToken) SetToken(w http.ResponseWriter, name, value string, options *sessions.Options) {
	http.SetCookie(w, sessions.NewCookie(name, value, options))
}
