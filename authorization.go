package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"time"
)

const (
	// Key used to identify cookies of type AuthorizedCookie.
	// Initialized in init().
	AUTHORIZED_COOKIE = "AuthorizedCookie"

	// How long to keep a user logged in after log in.
	// Initialized in init().
	LOGIN_DURATION = 15 * time.Minute
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

	loginExpiresOn := ac.AuthorizedOnUTC.Add(LOGIN_DURATION)
	authorized = time.Now().UTC().Before(loginExpiresOn)

	return authorized, nil
}

// Set a cookie on w that indicates that this user is logged in.
func SetAuthorized(w http.ResponseWriter) error {
	ac := AuthorizedCookie{
		AuthorizedOnUTC: time.Now().UTC(),
	}

	encoder := securecookie.New(HashKey, BlockKey)
	value, err := encoder.Encode(AUTHORIZED_COOKIE, &ac)
	if err != nil {
		return errors.Wrap(err, "could not encode cookie")
	}

	cookie := &http.Cookie{
		Name:     AUTHORIZED_COOKIE,
		Value:    value,
		Path:     "/",
		Secure:   false,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	return nil
}

// Return whether pass is a valid Passphrase defined in the configuration
// file.
func IsValidPassphrase(pass string) bool {
	ps := shasum(pass)

	for _, hs := range GetConfig().PassHashes {
		hb, err := hex.DecodeString(hs)
		if err != nil {
			log.Printf(`bad PassHashes entry="%v" in configuration file`, hs)
			continue
		}

		if eq(ps, hb) {
			return true
		}
	}

	return false
}

// Return the SHA256 sum of s.
func shasum(s string) []byte {
	h := sha256.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}

// Return whether s and t are equal.
func eq(s, t []byte) bool {
	return bytes.Compare(s, t) == 0
}
