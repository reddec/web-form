package engine

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/reddec/web-form/internal/notifications"
	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/utils"
	"github.com/reddec/web-form/internal/web"
)

const (
	accessCodeField = "accessCode"
	freshField      = "fresh"
	tzField         = "tz"
)

type Storage interface {
	Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error)
}

type WebhooksFactory interface {
	Create(webhook schema.Webhook) notifications.Notification
}

type AMQPFactory interface {
	Create(definition schema.AMQP) notifications.Notification
}

type FormConfig struct {
	Definition      schema.Form        // schema definition
	ViewForm        *template.Template // template to show main form
	ViewSuccess     *template.Template // template to show result (success) after submit
	ViewFail        *template.Template // template to show result (fail) after submit
	ViewCode        *template.Template // template to show code access
	ViewForbidden   *template.Template // template to show access denied
	Storage         Storage            // where to store data
	WebhooksFactory WebhooksFactory
	AMQPFactory     AMQPFactory
	XSRF            bool // check XSRF token. Disable if form is exposed as API.
	Captcha         []web.Captcha
}

func NewForm(config FormConfig, options ...FormOption) http.HandlerFunc {
	for _, opt := range options {
		opt(&config)
	}

	var destinations []notifications.Notification

	if config.WebhooksFactory != nil {
		for _, webhook := range config.Definition.Webhooks {
			destinations = append(destinations, config.WebhooksFactory.Create(webhook))
		}
	}

	if config.AMQPFactory != nil {
		for _, definition := range config.Definition.AMQP {
			destinations = append(destinations, config.AMQPFactory.Create(definition))
		}
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()

		f := &formRequest{
			FormConfig:   &config,
			destinations: destinations,
		}

		r := web.NewRequest(writer, request).WithCaptcha(config.Captcha...).Set("Form", &f.Definition)
		f.Serve(r)
	}
}

type formRequest struct {
	*FormConfig
	destinations []notifications.Notification
}

//nolint:cyclop
func (fr *formRequest) Serve(request *web.Request) {
	// check XSRF tokens (POST only)
	if fr.XSRF && !request.VerifyXSRF() {
		request.Error("XSRF validation failed")
		request.Render(http.StatusForbidden, fr.ViewForbidden)
		return
	}

	// check credentials access (OIDC)
	if !fr.Definition.IsAllowed(request.Credentials()) {
		request.Render(http.StatusForbidden, fr.ViewForbidden)
		return
	}

	// for code access forms, all interactions should be done via POST
	if fr.Definition.HasCodeAccess() && request.Request().Method != http.MethodPost {
		request.Render(http.StatusUnauthorized, fr.ViewCode)
		return
	}

	// check code access
	if !validateCode(&fr.Definition, request) {
		request.Error("invalid code")
		request.Render(http.StatusUnauthorized, fr.ViewCode)
		return
	}

	// pre-render default values
	if err := fr.preRender(request); err != nil {
		request.Error("render defaults " + err.Error())
	}

	// if it's fresh start - show page without processing data
	if request.Pop(freshField) == "true" || request.Request().Method == http.MethodGet {
		request.Render(http.StatusOK, fr.ViewForm)
		return
	}

	// check captcha (form post only)
	if !request.VerifyCaptcha() {
		request.Error("invalid captcha")
		request.Render(http.StatusBadRequest, fr.ViewForm)
		return
	}

	// it's not fresh start or get - submit the form
	fr.submitForm(request)
}

func (fr *formRequest) submitForm(request *web.Request) {
	tz := request.Session()[tzField]
	tzLocation, err := time.LoadLocation(tz)
	if err != nil {
		request.Logger().Warn("failed load client's timezone location - local will be used", "tz", tz, "error", err)
		tzLocation = time.Local
	}

	_ = request.Request().FormValue("") // parse form using Go defaults

	values, fieldErrors := schema.ParseForm(&fr.Definition, tzLocation, newRequestContext(request))

	// save flash messages with name related to field name
	for _, fieldError := range fieldErrors {
		request.Flash(fieldError.Name, fieldError.Error, web.FlashError)
	}

	if len(fieldErrors) > 0 {
		request.Logger().Info("form validation failed", toLogErrors(fieldErrors)...)
		request.Render(http.StatusUnprocessableEntity, fr.ViewForm)
		return
	}

	// bellow we will show success or failed page
	request.Push(freshField, "true")
	result, storeErr := fr.Storage.Store(request.Context(), fr.Definition.Table, values)
	if storeErr != nil {
		request.Error("failed to store data")
		request.Set("Result", &schema.ResultContext{
			Form:   &fr.Definition,
			Result: result,
			Error:  storeErr,
		})
		request.Render(http.StatusInternalServerError, fr.ViewFail)
		return
	}

	request.Set("Result", &schema.ResultContext{
		Form:   &fr.Definition,
		Result: result,
	}).Render(http.StatusOK, fr.ViewSuccess)

	fr.sendNotifications(request, schema.NotifyContext{
		Form:   &fr.Definition,
		Result: result,
	})
}

func (fr *formRequest) sendNotifications(request *web.Request, rc schema.NotifyContext) {
	ctx := request.Context()
	// send all notifications in parallel to avoid blocking in case one of dispatcher is slow/full
	var wg sync.WaitGroup

	for _, notify := range fr.destinations {
		notify := notify
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := notify.Dispatch(ctx, rc); err != nil {
				request.Logger().Error("failed dispatch notification", "error", err)
			}
		}()
	}

	wg.Wait()
}

func (fr *formRequest) preRender(request *web.Request) error {
	var defaultValues = make(map[string]any, len(fr.Definition.Fields))
	rct := newRequestContext(request)

	description, err := fr.Definition.Description.String(rct)
	if err != nil {
		return fmt.Errorf("render description: %w", err)
	}
	request.Set("Description", description)

	for _, field := range fr.Definition.Fields {

		// if there is old value - keep it as default
		if oldValues := request.Request().Form[field.Name]; len(oldValues) > 0 {
			// multiselect is special case - we need to preserve options
			if field.Multiple && len(field.Options) > 0 {
				defaultValues[field.Name] = utils.NewSet(oldValues...)
			} else {
				defaultValues[field.Name] = oldValues[0]
			}
		} else {
			// fresh default value
			value, err := field.Default.String(rct)
			if err != nil {
				return fmt.Errorf("compute default value for field %q: %w", field.Name, err)
			}
			defaultValues[field.Name] = value
		}
	}
	request.Set("Defaults", defaultValues)
	return nil
}

func newRequestContext(request *web.Request) *schema.RequestContext {
	return &schema.RequestContext{
		Headers:     request.Request().Header,
		Query:       request.Request().URL.Query(),
		Form:        request.Request().PostForm,
		Code:        request.Session()[accessCodeField],
		Credentials: request.Credentials(),
	}
}

func validateCode(form *schema.Form, request *web.Request) bool {
	if !form.HasCodeAccess() {
		return true
	}

	// only post allowed
	if request.Request().Method != http.MethodPost {
		return false
	}

	// check session value
	code := request.Session()[accessCodeField]
	if code == "" {
		// maybe it's a form post
		code = request.Request().PostFormValue(accessCodeField)
		request.Push(freshField, "true")
	}

	if !form.Codes.Has(code) {
		return false
	}
	request.Session()[accessCodeField] = code // save code in order to re-use
	request.Set("code", code)                 // for UI
	return true
}

func toLogErrors(fieldError []schema.FieldError) []any {
	var ans = make([]any, 0, 2*len(fieldError))
	for _, f := range fieldError {
		ans = append(ans, "field."+f.Name, f.Error)
	}
	return ans
}
