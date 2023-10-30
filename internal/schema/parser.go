package schema

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/reddec/web-form/internal/utils"
)

var (
	ErrInvalidType   = errors.New("field type invalid")
	ErrRequiredField = errors.New("required field not set")
	ErrWrongPattern  = errors.New("doesn't match pattern")
)

func (t *Type) UnmarshalText(text []byte) error {
	v := Type(text)
	switch v {
	case "":
		*t = TypeString
	case TypeString, TypeBoolean, TypeFloat, TypeInteger, TypeDate, TypeDateTime:
		*t = v
	default:
		return fmt.Errorf("field type %q: %w", v, ErrInvalidType)
	}
	return nil
}

func (t Type) Parse(value string, locale *time.Location) (any, error) {
	switch t {
	case TypeString:
		return value, nil
	case TypeInteger:
		return strconv.ParseInt(value, 10, 64)
	case TypeFloat:
		return strconv.ParseFloat(value, 64)
	case TypeBoolean:
		return strconv.ParseBool(value)
	case TypeDate:
		return time.ParseInLocation("2006-01-02", value, locale)
	case TypeDateTime:
		return time.ParseInLocation("2006-01-02T15:04", value, locale) // html type
	default:
		return value, nil
	}
}

func (f *Field) Parse(value string, locale *time.Location, viewCtx *RequestContext) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		v, err := f.Default.String(viewCtx)
		if err != nil {
			return nil, fmt.Errorf("render default value: %w", err)
		}
		value = v
	}
	if value == "" && f.Required {
		return nil, ErrRequiredField
	}

	if (f.Type == "" || f.Type == TypeString) && f.Pattern != "" {
		if ok, _ := regexp.MatchString(f.Pattern, value); !ok {
			return nil, fmt.Errorf("%w: %q", ErrWrongPattern, f.Pattern)
		}
	}

	return f.Type.Parse(value, locale)
}

func OptionValues(options ...Option) utils.Set[string] {
	var ans = make([]string, 0, len(options))
	for _, opt := range options {
		if opt.Value != "" {
			ans = append(ans, opt.Value)
		} else {
			ans = append(ans, opt.Label)
		}
	}
	return utils.NewSet(ans...)
}

func (p *Policy) UnmarshalText(text []byte) error {
	env, err := cel.NewEnv(
		cel.Variable("user", cel.StringType),
		cel.Variable("email", cel.StringType),
		cel.Variable("groups", cel.ListType(cel.StringType)),
	)
	if err != nil {
		return fmt.Errorf("create CEL env: %w", err)
	}
	ast, issues := env.Compile(string(text))
	if issues != nil {
		return fmt.Errorf("parse policy %q: %w", string(text), issues.Err())
	}
	prog, err := env.Program(ast)
	if err != nil {
		return fmt.Errorf("compile CEL AST %q: %w", string(text), err)
	}
	p.Program = prog
	return nil
}

type credsKey struct{}

func WithCredentials(ctx context.Context, creds *Credentials) context.Context {
	return context.WithValue(ctx, credsKey{}, creds)
}

func CredentialsFromContext(ctx context.Context) *Credentials {
	c, _ := ctx.Value(credsKey{}).(*Credentials)
	return c
}

// ParseForm converts user request to parsed field.
//
//nolint:cyclop
func ParseForm(definition *Form, tzLocation *time.Location, viewCtx *RequestContext) (map[string]any, []FieldError) {
	var fields = make(map[string]any, len(definition.Fields))
	var fieldErrors []FieldError

	for _, field := range definition.Fields {
		field := field

		var values []string
		if field.Hidden || field.Disabled {
			// if field is non-accessible by user we shall ignore user input and use default as value
			v, err := field.Default.String(viewCtx)
			if err != nil {
				// super abnormal situation - default field failed so it's unfixable by user until configuration update
				fieldErrors = append(fieldErrors, FieldError{
					Name:  field.Name,
					Error: err,
				})
				continue
			}
			values = append(values, v)
		} else {
			// collect all user input (could be more than one in case of array)
			values = utils.Uniq(viewCtx.Form[field.Name])
		}

		if len(field.Options) > 0 {
			// we need to check that all values belong to allowed values before parsing
			// since we are using plain text comparison.
			options := OptionValues(field.Options...)
			if !options.Has(values...) {
				fieldErrors = append(fieldErrors, FieldError{
					Name:  field.Name,
					Error: errors.New("selected not allowed option"),
				})
				continue
			}
		}

		parsedValues, err := parseValues(values, &field, tzLocation, viewCtx)
		if err != nil {
			fieldErrors = append(fieldErrors, FieldError{
				Name:  field.Name,
				Error: err,
			})
			continue
		}

		if field.Required && len(parsedValues) == 0 {
			// corner case - empty array for required field. Can happen if multi-select and nothing selected.
			// for empty values it will be checked by field parser.
			//
			// We also need parse first to exclude empty values.
			fieldErrors = append(fieldErrors, FieldError{
				Name:  field.Name,
				Error: errors.New("required field is not provided"),
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

func parseValues(values []string, field *Field, tzLocation *time.Location, viewCtx *RequestContext) ([]any, error) {
	var ans = make([]any, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) == 0 {
			continue
		}
		v, err := field.Parse(value, tzLocation, viewCtx)
		if err != nil {
			return nil, err
		}
		ans = append(ans, v)
	}
	return ans, nil
}

type FieldError struct {
	Name  string
	Error error
}
