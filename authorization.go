package main

import (
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"
)

const (
	// Key used to identify cookies of type AuthorizedCookie.
	AUTHORIZED_COOKIE = "AuthorizedCookie"

	// How long to keep a user logged in after log in.
	LOGIN_DURATION = 14 * 24 * time.Hour
)

var (
	// The global hash key used for use with securecookie.
	HashKey []byte

	// The global block key for use with securecookie.
	BlockKey []byte
)

// The cookie we set if a user successfull logs in.
type AuthorizedCookie struct {
	// When this cookie was created created.
	AuthorizedOnUTC time.Time

	// Whether this user is actually logged in.
	//
	// When a user logs in, a user with LoggedIn set to
	// true is set. When the user logs out, a cookie with
	// LoggedIn set to false is set instead. We do not only
	// clear the cookie to avoid annoying error messages when
	// parsing the cookie.
	LoggedIn bool
}

// Return the expiry date for the given cookie. This function ignores the value
// of LoggedIn, that is it simply returns AuthorizedOnUTC incremented with the
// fixed login duration.
func (ac *AuthorizedCookie) ExpiryDate() time.Time {
	return ac.AuthorizedOnUTC.Add(LOGIN_DURATION)
}

// Return whether user with cookie c is authorized to access
// restricted resource at this current time.
func (ac *AuthorizedCookie) Authorized() bool {
	expired := time.Now().UTC().After(ac.ExpiryDate())
	return ac.LoggedIn && !expired
}

// Initialize HashKey and BlockKey. We do not persist these keys, after
// re-starting all logged in users will have to log in again.
func init() {
	if HashKey = securecookie.GenerateRandomKey(64); HashKey == nil {
		log.Fatal("could not generate cookie hash key")
	}

	if BlockKey = securecookie.GenerateRandomKey(32); BlockKey == nil {
		log.Fatal("could not generate cookie block key")
	}
}

// Return whether request r is authenticated to upload, delete and
// list files.
func IsAuthorized(r *http.Request) (authorized bool, err error) {
	cookie, err := r.Cookie(AUTHORIZED_COOKIE)
	if err != nil {
		return false, errors.Wrap(err, "could not read cookie from request")
	}

	var ac AuthorizedCookie

	encoder := securecookie.New(HashKey, BlockKey)
	if err := encoder.Decode(AUTHORIZED_COOKIE, cookie.Value, &ac); err != nil {
		return false, errors.Wrap(err, "could not decode cookie")
	}

	return ac.Authorized(), nil
}

// Set a cookie on w that indicates that this user is logged in.
func SetAuthorized(w http.ResponseWriter) error {
	ac := AuthorizedCookie{
		AuthorizedOnUTC: time.Now().UTC(),
		LoggedIn:        true,
	}

	return setCookie(w, &ac)
}

// Set a cookie on w that indicates that this user is not logged in.
func SetUnauthorized(w http.ResponseWriter) error {
	ac := AuthorizedCookie{
		AuthorizedOnUTC: time.Now().UTC(),
		LoggedIn:        false,
	}

	return setCookie(w, &ac)
}

// Return whether pass is a valid password set up in the configuration
// file.
func IsValidPassword(pass string) bool {
	pb := []byte(pass)

	for _, hs := range GetConfig().PassHashes {
		hb := []byte(hs)

		if err := bcrypt.CompareHashAndPassword(hb, pb); err == nil {
			return true
		}
	}

	return false
}

func setCookie(w http.ResponseWriter, ac *AuthorizedCookie) error {
	encoder := securecookie.New(HashKey, BlockKey)
	value, err := encoder.Encode(AUTHORIZED_COOKIE, &ac)
	if err != nil {
		return errors.Wrap(err, "could not encode cookie")
	}

	cookie := &http.Cookie{
		Expires: ac.ExpiryDate(),
		HttpOnly: true,
		Name:     AUTHORIZED_COOKIE,
		Path:     "/",
		Secure:   false,
		Value:    value,
	}

	http.SetCookie(w, cookie)
	return nil
}
