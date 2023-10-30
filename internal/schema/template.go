package schema

import (
	"bytes"
	"net/http"
	"net/url"
	"text/template"

	"github.com/reddec/web-form/internal/utils"
)

func MustTemplate[T any](text string) Template[T] {
	v, err := NewTemplate[T](text)
	if err != nil {
		panic(err)
	}
	return v
}

func NewTemplate[T any](text string) (Template[T], error) {
	v, err := template.New("").Funcs(utils.TemplateFuncs()).Parse(text)
	if err != nil {
		return Template[T]{}, err
	}
	return Template[T]{Value: v, Valid: true}, nil
}

type Template[T any] struct {
	Valid bool
	Value *template.Template
}

func (t *Template[T]) UnmarshalText(text []byte) error {
	v, err := NewTemplate[T](string(text))
	if err != nil {
		return err
	}
	*t = v
	return nil
}

func (t *Template[T]) Bytes(data *T) ([]byte, error) {
	if !t.Valid {
		return nil, nil
	}
	var buf bytes.Buffer
	err := t.Value.Execute(&buf, data)
	return buf.Bytes(), err
}

func (t *Template[T]) String(data *T) (string, error) {
	if !t.Valid {
		return "", nil
	}
	v, err := t.Bytes(data)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// RequestContext is used for rendering default values.
type RequestContext struct {
	Headers     http.Header
	Query       url.Values
	Form        url.Values
	Code        string       // access code
	Credentials *Credentials // optional user credentials
}

func (rc *RequestContext) User() string {
	if rc.Credentials == nil {
		return ""
	}
	return rc.Credentials.User
}

func (rc *RequestContext) Groups() []string {
	if rc.Credentials == nil {
		return nil
	}
	return rc.Credentials.Groups
}

func (rc *RequestContext) Email() string {
	if rc.Credentials == nil {
		return ""
	}
	return rc.Credentials.Email
}

// ResultContext is used for rendering result message (success or fail).
type ResultContext struct {
	Form   *Form
	Result map[string]any
	Error  error
}

// NotifyContext is used for rendering notification message.
type NotifyContext struct {
	Form   *Form
	Result map[string]any
}
