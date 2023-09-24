package schema

import (
	"html/template"
	"time"
)

func Default() Form {
	return Form{
		Success: "Thank you for the submission!",
		Failed:  "Something went wrong: `{{.Error}}`",
	}
}

type Template template.Template

type Form struct {
	Name        string // unique form name, if not set - file name without extension will be used.
	Table       string // database table name
	Title       string // optional title for the form
	Description string // (markdown) optional description of the form
	Fields      []Field
	Webhooks    []Webhook
	Success     string // markdown message for success (also go template with available .Result)
	Failed      string // markdown message for failed (also go template with .Error)
}

type Type string

const (
	TypeString   Type = "string" // default, also for enums
	TypeInteger  Type = "integer"
	TypeFloat    Type = "float"
	TypeBoolean  Type = "boolean"
	TypeDate     Type = "date"
	TypeDateTime Type = "date-time"
)

func (t Type) Is(value string) bool {
	// special case
	if value == "number" {
		return t == TypeInteger || t == TypeFloat
	}
	if t == "" {
		return value == "string"
	}
	return Type(value) == t
}

type Field struct {
	Name        string   // column name in database.
	Label       string   // short name of field which will be shown in UI, if not set - [Field.Name] is used.
	Description string   // (markdown) optional description for the field, also shown in UI as help text.
	Required    bool     // make field as required: empty values will not be accepted as well as at least one option should be selected.
	Disabled    bool     // user input will be ignored, by field will be visible in UI. Doesn't apply for options.
	Hidden      bool     // user input will be ignored, field not visible in UI
	Default     string   // golang template expression for the default value.  Doesn't apply for options with [Field.Multiple].
	Type        Type     // (default [TypeString]) field type used for user input validation.
	Pattern     string   // optional regexp to validate content, applicable only for string type
	Options     []Option // allowed values. If [Field.Multiple] set, it acts as "any of", otherwise "one of".
	Multiple    bool     // allow picking multiple options. Column type in database MUST be ARRAY of corresponding type.
	Multiline   bool     // multiline input (for [TypeString] only)
}

type Webhook struct {
	URL      string            // URL for POST webhook, where payload is JSON with fields from database column.
	Method   string            // HTTP method to perform, default is POST
	Retry    int               // maximum number of retries (0 or negative means no retries)
	Timeout  time.Duration     // request timeout
	Interval time.Duration     // interval between attempts (for non 2xx code)
	Headers  map[string]string // arbitrary headers (ex: Authorization)
	Message  *Template         // payload content, if not set - JSON representation of storage result
}

type Option struct {
	Label string // label for UI
	Value string // if not set - Label is used, allowed value should match textual representation of form value
}