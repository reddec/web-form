package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/utils"

	oidclogin "github.com/reddec/oidc-login"
)

type Storage interface {
	Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error)
}

type WebhooksDispatcher interface {
	Dispatch(ctx context.Context, webhook schema.Webhook, payload []byte) error
}

type FormConfig struct {
	Definition         schema.Form        // schema definition
	Renderer           *Renderer          // renderer for template blocks
	ViewForm           *template.Template // template to show main form
	ViewResult         *template.Template // template to show result after submit
	Storage            Storage            // where to store data
	WebhooksDispatcher WebhooksDispatcher // how to execute webhooks
	XSRF               bool               // check XSRF token. Disable if form is exposed as API.
}

func NewForm(config FormConfig, options ...FormOption) *Form {
	for _, opt := range options {
		opt(&config)
	}
	return &Form{
		config: config,
	}
}

type Form struct {
	config FormConfig
}

func (f *Form) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	switch request.Method {
	case http.MethodGet:
		f.renderForm(http.StatusOK, writer, request, nil)
		return
	case http.MethodPost:
		if f.config.XSRF && !verifyXSRF(request) {
			slog.Warn("XSRF verification failed")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		f.submitForm(writer, request)
		return
	default:
		http.Error(writer, "only POST or GET supported", http.StatusMethodNotAllowed)
	}
}

func (f *Form) renderForm(code int, writer http.ResponseWriter, request *http.Request, fieldErrors []fieldError) {
	vc := &ViewContext{
		Form:   f.config.Definition,
		View:   f.config.Renderer.View(request),
		XSRF:   XSRF(writer),
		Errors: fieldErrors,
	}

	var buffer bytes.Buffer
	err := f.config.ViewForm.Execute(&buffer, vc)
	if err != nil {
		slog.Error("failed render form", "form", f.config.Definition.Title, "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(code)
	_, _ = writer.Write(buffer.Bytes())
}

func (f *Form) submitForm(writer http.ResponseWriter, request *http.Request) {
	tz := request.FormValue("__tz")

	tzLocation, err := time.LoadLocation(request.FormValue("__tz"))
	if err != nil {
		slog.Warn("failed load client's timezone location - local will be used", "tz", tz, "error", err)
		tzLocation = time.Local
	}

	values, fieldErrors := f.parseFields(request, tzLocation)
	if len(fieldErrors) > 0 {
		slog.Info("form validation failed", "form", f.config.Definition.Name)
		f.renderForm(http.StatusUnprocessableEntity, writer, request, fieldErrors)
		return
	}

	result, storeErr := f.config.Storage.Store(request.Context(), f.config.Definition.Table, values)
	if storeErr != nil {
		slog.Error("failed store data", "error", storeErr)
	}

	rc := &ResultContext{
		req:    request,
		render: f.config.Renderer,
		Form:   f.config.Definition,
		Error:  storeErr,
		Result: result,
	}

	var buffer bytes.Buffer
	err = f.config.ViewResult.Execute(&buffer, rc)
	if err != nil {
		slog.Error("failed render result view", "form", f.config.Definition.Title, "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html")
	if storeErr != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusCreated)
		if err := f.sendWebhooks(request.Context(), rc); err != nil {
			slog.Error("failed send webhooks", "error", err)
		}
	}
	_, _ = writer.Write(buffer.Bytes())
}

func (f *Form) sendWebhooks(ctx context.Context, rc *ResultContext) error {
	for _, wh := range f.config.Definition.Webhooks {
		wh := wh
		payload, err := renderWebhook(&wh, rc)
		if err != nil {
			slog.Error("failed render payload for webhook", "form", f.config.Definition.Title, "webhook", wh.URL, "error", err)
			continue
		}

		if err := f.config.WebhooksDispatcher.Dispatch(ctx, wh, payload); err != nil {
			return fmt.Errorf("dispatch webhook: %w", err)
		}
	}
	return nil
}

//nolint:cyclop
func (f *Form) parseFields(request *http.Request, tzLocation *time.Location) (map[string]any, []fieldError) {
	view := f.config.Renderer.View(request)

	var fields = make(map[string]any, len(f.config.Definition.Fields))
	var fieldErrors []fieldError
	for _, field := range f.config.Definition.Fields {
		field := field

		var values []string
		if field.Hidden || field.Disabled {
			v, err := view.Render(field.Default)
			if err != nil {
				// super abnormal situation - default field failed
				fieldErrors = append(fieldErrors, fieldError{
					Field: field.Name,
					Error: err,
				})
				continue
			}
			values = append(values, v)
		} else {
			values = utils.Uniq(request.Form[field.Name])
		}

		if len(field.Options) > 0 {
			// we need to check that all values belong to allowed values before parsing
			// since we are using plain text comparison.
			options := schema.OptionValues(field.Options...)
			if !options.Has(values...) {
				fieldErrors = append(fieldErrors, fieldError{
					Field: field.Name,
					Error: errors.New("selected not allowed option"),
				})
				continue
			}
		}

		parsedValues, err := parseValues(values, &field, tzLocation, view)
		if err != nil {
			fieldErrors = append(fieldErrors, fieldError{
				Field: field.Name,
				Error: err,
			})
			continue
		}

		if field.Required && len(parsedValues) == 0 {
			// corner case - empty array for required field. Can happen if multi-select and nothing selected.
			// for empty values it will be checked by field parser.
			//
			// We also need parse first to exclude empty values.
			fieldErrors = append(fieldErrors, fieldError{
				Field: field.Name,
				Error: errors.New("at least one option should be selected"),
			})
			continue
		}

		if field.Multiple {
			fields[field.Name] = parsedValues
		} else if len(parsedValues) > 0 {
			fields[field.Name] = parsedValues[0]
		}
	}
	return fields, fieldErrors
}

func parseValues(values []string, field *schema.Field, tzLocation *time.Location, view *View) ([]any, error) {
	var ans = make([]any, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		v, err := field.Parse(value, tzLocation, view)
		if err != nil {
			return nil, err
		}
		ans = append(ans, v)
	}
	return ans, nil
}

type fieldError struct {
	Field string
	Error error
}

type ResultContext struct {
	req    *http.Request
	render *Renderer
	Form   schema.Form
	Error  error
	Result map[string]any
}

func (rc *ResultContext) Render(value string) (string, error) {
	return rc.render.Render(value, rc.req, rc.Result, rc.Error)
}

type ViewContext struct {
	*View
	XSRF   string
	Form   schema.Form
	Errors []fieldError
}

func (vc *ViewContext) LastValue(name string) string {
	if vc.req.Method == http.MethodPost {
		return vc.req.FormValue(name)
	}
	return ""
}

func (vc *ViewContext) LastValues(name string) utils.Set[string] {
	if vc.req.Method == http.MethodPost {
		return utils.NewSet(vc.req.Form[name]...)
	}
	return utils.NewSet[string]()
}

func (vc *ViewContext) FieldError(name string) error {
	for _, f := range vc.Errors {
		if f.Field == name {
			return f.Error
		}
	}
	return nil
}

type Renderer struct {
	cache sync.Map // string -> template
	funcs template.FuncMap
}

// Render single value as golang template. Caches parsed template.
func (r *Renderer) Render(value string, req *http.Request, result map[string]any, dataErr error) (string, error) {
	t, err := r.getOrCompute(value)
	if err != nil {
		return "", fmt.Errorf("get template: %w", err)
	}

	if err := req.ParseForm(); err != nil {
		return "", fmt.Errorf("parse form: %w", err)
	}

	token := oidclogin.Token(req)

	vc := renderContext{
		Headers: req.Header,
		Query:   req.URL.Query(),
		Form:    req.Form,
		Result:  result,
		Error:   dataErr,
		User:    oidclogin.User(token),
		Email:   oidclogin.Email(token),
		Groups:  oidclogin.Groups(token),
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, &vc)
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}

func (r *Renderer) getOrCompute(value string) (*template.Template, error) {
	v, ok := r.cache.Load(value)
	if ok {
		return v.(*template.Template), nil //nolint:forcetypeassert
	}
	t, err := template.New("").Funcs(r.funcs).Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	r.cache.Store(value, t)
	return t, nil
}

func (r *Renderer) View(req *http.Request) *View {
	return &View{
		render: r,
		req:    req,
	}
}

type renderContext struct {
	Headers http.Header
	Query   url.Values
	Form    url.Values
	Result  map[string]any
	Error   error
	User    string
	Groups  []string
	Email   string
}

type View struct {
	render *Renderer
	req    *http.Request
}

func (v *View) Render(value string) (string, error) {
	return v.render.Render(value, v.req, nil, nil)
}

func renderWebhook(wh *schema.Webhook, rc *ResultContext) ([]byte, error) {
	if wh.Message == nil {
		return json.Marshal(rc.Result)
	}
	return wh.Message.Render(rc)
}
