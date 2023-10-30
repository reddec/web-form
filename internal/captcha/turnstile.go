package captcha

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/reddec/web-form/internal/web"
)

type Turnstile struct {
	SiteKey   string        `long:"site-key" env:"SITE_KEY" description:"Widget access key"`
	SecretKey string        `long:"secret-key" env:"SECRET_KEY" description:"Server side secret key"`
	Timeout   time.Duration `long:"timeout" env:"TIMEOUT" description:"Validation request timeout" default:"3s"`
}

func (widget *Turnstile) Embed() template.HTML {
	//nolint:gosec
	return template.HTML(`
   <div class="cf-turnstile" data-sitekey="` + url.QueryEscape(widget.SiteKey) + `"></div>
   <script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>
`)
}

func (widget *Turnstile) Validate(form *http.Request) bool {
	const validateURL = `https://challenges.cloudflare.com/turnstile/v0/siteverify`
	var response struct {
		Success bool `json:"success"`
	}
	ctx, cancel := context.WithTimeout(form.Context(), widget.Timeout)
	defer cancel()

	var formFields = make(url.Values)
	formFields.Add("secret", widget.SecretKey)
	formFields.Add("response", form.Form.Get("cf-turnstile-response"))
	formFields.Add("remoteip", web.GetClientIP(form))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validateURL, strings.NewReader(formFields.Encode()))
	if err != nil {
		slog.Error("failed create request to check turnstile captcha", "error", err)
		return false
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed execute request to check turnstile captcha", "error", err)
		return false
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		slog.Error("failed decode response from turnstile captcha", "error", err)
		return false
	}

	return response.Success
}
