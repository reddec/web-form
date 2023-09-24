package schema

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sql-web-form/internal/utils"
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

func (f *Field) Parse(value string, locale *time.Location, render interface{ Render(string) (string, error) }) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		v, err := render.Render(f.Default)
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

func (t *Template) UnmarshalText(text []byte) error {
	v, err := template.New("").Funcs(utils.TemplateFuncs()).Parse(string(text))
	if err != nil {
		return err
	}
	*t = Template(*v)
	return nil
}

func (t *Template) Render(data any) ([]byte, error) {
	var buf bytes.Buffer
	err := (*template.Template)(t).Execute(&buf, data)
	return buf.Bytes(), err
}
