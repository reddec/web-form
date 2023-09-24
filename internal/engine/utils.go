package engine

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
)

const (
	FormXSRF   = "_xsrf"
	CookieXSRF = "_xsrf"
)

// XSRF protection token. Returned token should be submitted as _xsrf form value. Panics if crypto generator is not available.
func XSRF(writer http.ResponseWriter) string {
	var token [32]byte
	_, err := io.ReadFull(rand.Reader, token[:])
	if err != nil {
		panic(err)
	}

	t := hex.EncodeToString(token[:])
	http.SetCookie(writer, &http.Cookie{
		Name:     CookieXSRF,
		Value:    t,
		Path:     ".",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return t
}

// verify XSRF token from cookie and form and removes it from form values.
func verifyXSRF(req *http.Request) bool {
	cookie, err := req.Cookie(CookieXSRF)
	if err != nil {
		return false
	}
	formValue := req.FormValue(FormXSRF)
	req.Form.Del(FormXSRF)
	return cookie.Value == formValue && formValue != ""
}
