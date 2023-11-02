package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/utils"
)

const (
	FormXSRF   = "_xsrf"
	CookieXSRF = "_xsrf"
)

type Captcha interface {
	Embed() template.HTML
	Validate(form *http.Request) bool
}

func NewRequest(writer http.ResponseWriter, request *http.Request) *Request {
	remoteIP := GetClientIP(request)

	var params = []any{
		"path", request.URL.Path,
		"method", request.Method,
		"remote_ip", remoteIP,
	}
	creds := schema.CredentialsFromContext(request.Context())
	if creds != nil {
		params = append(params, "user", creds.User)
	}

	return &Request{
		logger:  slog.With(params...),
		writer:  writer,
		request: request,
		creds:   creds,
	}
}

type Request struct {
	messages []FlashMessage
	xsrf     string
	writer   http.ResponseWriter
	request  *http.Request
	logger   *slog.Logger
	state    map[string]any
	session  map[string]string
	creds    *schema.Credentials
	captchas []Captcha
}

func (r *Request) VerifyCaptcha() bool {
	if r.request.Method != http.MethodPost {
		return true
	}
	for _, captcha := range r.captchas {
		if !captcha.Validate(r.request) {
			return false
		}
	}
	return true
}

func (r *Request) VerifyXSRF() bool {
	return r.request.Method != http.MethodPost || verifyXSRF(r.request)
}

func (r *Request) WithCaptcha(captcha ...Captcha) *Request {
	r.captchas = captcha
	return r
}

func (r *Request) Request() *http.Request {
	return r.request
}

func (r *Request) Context() context.Context {
	return r.request.Context()
}

func (r *Request) Logger() *slog.Logger {
	return r.logger
}

func (r *Request) Credentials() *schema.Credentials {
	return r.creds
}

// Clear session.
func (r *Request) Clear() {
	r.session = nil
}

// Session values. Visible to clients, but available only via POST.
func (r *Request) Session() map[string]string {
	if r.session == nil {
		r.session = r.parseSession()
	}
	return r.session
}

// Pop value from session.
func (r *Request) Pop(sessionKey string) string {
	v := r.session[sessionKey]
	delete(r.session, sessionKey)
	return v
}

// Push value to session.
func (r *Request) Push(sessionKey string, value string) *Request {
	r.Session()[sessionKey] = value
	return r
}

func (r *Request) Set(key string, state any) *Request {
	if r.state == nil {
		r.state = make(map[string]any)
	}
	r.state[key] = state
	return r
}

func (r *Request) State() map[string]any {
	return r.state
}

// Error flash message.
func (r *Request) Error(message any) *Request {
	r.Flash("", message, FlashError)
	return r
}

// Info flash message.
func (r *Request) Info(message any) *Request {
	r.Flash("", message, FlashInfo)
	return r
}

func (r *Request) Flash(name string, message any, flashType FlashType) *Request {
	r.messages = append(r.messages, FlashMessage{
		Text: fmt.Sprint(message),
		Type: flashType,
		Name: name,
	})
	return r
}

func (r *Request) Messages(names ...string) []FlashMessage {
	if len(names) == 0 {
		return r.messages
	}
	idx := utils.NewSet(names...)
	var ans []FlashMessage
	for _, f := range r.messages {
		if idx.Has(f.Name) {
			ans = append(ans, f)
		}
	}
	return ans
}

func (r *Request) EmbedXSRF() template.HTML {
	if r.xsrf == "" {
		r.xsrf = XSRF(r.writer)
	}
	return template.HTML(`<input type="hidden" name="` + FormXSRF + `" value="` + r.xsrf + `"/>`) //nolint:gosec
}

func (r *Request) EmbedSession() template.HTML {
	var out string
	for k, v := range r.session {
		out += `<input type="hidden" name="__` + url.QueryEscape(k) + `" value="` + url.QueryEscape(v) + `"/>`
	}
	return template.HTML(out) //nolint:gosec
}

func (r *Request) EmbedCaptcha() template.HTML {
	var out string
	for _, v := range r.captchas {
		out += string(v.Embed())
	}
	return template.HTML(out) //nolint:gosec
}

func (r *Request) Render(code int, view *template.Template) {
	var buffer bytes.Buffer
	err := view.Execute(&buffer, r)
	if err != nil {
		slog.Error("failed render", "error", err, "path", r.request.URL.Path, "method", r.request.Method)
		r.writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.writer.Header().Set("Content-Type", "text/html")
	r.writer.WriteHeader(code)
	_, _ = r.writer.Write(buffer.Bytes())
}

func (r *Request) parseSession() map[string]string {
	_ = r.request.PostFormValue("") // let Go parse form properly

	var session = make(map[string]string)
	for k := range r.request.PostForm {
		if strings.HasPrefix(k, "__") {
			session[k[2:]] = r.request.PostForm.Get(k)
		}
	}

	return session
}

type FlashType string

const (
	FlashError FlashType = "danger"
	FlashInfo  FlashType = "info"
)

type FlashMessage struct {
	Name string
	Text string
	Type FlashType
}

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
		Path:     "/", // we have to set cookie on root due to iOS limitations
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour),
	})
	return t
}

// verify XSRF token from cookie and form.
func verifyXSRF(req *http.Request) bool {
	cookie, err := req.Cookie(CookieXSRF)
	if err != nil {
		return false
	}
	formValue := req.FormValue(FormXSRF)
	return cookie.Value == formValue && formValue != ""
}

func GetClientIP(r *http.Request) string {
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		for _, part := range parts {
			return strings.TrimSpace(part)
		}
	}
	return r.RemoteAddr
}
