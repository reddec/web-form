# Fields
<!--  {% raw %} --> 

Only field `name` is required for the field.

For all fields text values will be trimmed from leading and trailing white spaces.

Configurations:

| Name          | Type                | Default  | Description                                                         |
|---------------|---------------------|----------|---------------------------------------------------------------------|
| *`name`*      | string              |          | column name in database and unique identifier of the field          |
| `label`       | string              | `.name`  | name of field if UI                                                 |
| `description` | string              |          | short description for the field, will be shown in UI as hint        |
| `required`    | boolean             | false    | is field required                                                   |
| `disabled`    | boolean             | false    | is user allowed to edit field                                       |
| `hidden`      | boolean             | false    | do not show field in UI. Implicitly disables field                  |
| `default`     | string              |          | default value for the field. Supports [template](template.md)       |
| `type`        | [type](#types)      | `string` | field type                                                          |
| `pattern`     | string              |          | validate user input by regular expression (only for `string` types) |
| `options`     | [][Option](#option) |          | enum of allowed values                                              |
| `multiple`    | boolean             | false    | allow multiple options                                              |
| `multiline`   | boolean             | false    | tell UI to show multi-line input. Has no effect for backend         |

Notes:

- `required` fields can not be empty
- `default` field can be used for [prefill](prefill.md)
- `description` supports markdown and supports [templating](template.md#context-for-defaults)
- if `options` is set then:
    - with `multiple: true` it acts as "any of" (multiple choice)
    - otherwise it acts "one of" (single pick)
- if `multiple` is true, depending on [storage](stores.md), information will be stored as array or as plain string

The system has minimal trust to user input therefore:

- `hidden` or `disabled` fields are ignored even if it was provided in POST request
- `type` and `pattern` verification will be additionally checked on backend side


**Examples**

```yaml
- name: delivery_date
  label: When to deliver
  default: '{{now | date "2006-01-02T15:04"}}'
  required: true
  type: date-time


- name: birthday
  label: Your birthday
  default: '{{now | date "2006-01-02"}}'
  description: We will give you a discount
  type: date

- name: client_id
  label: Customer
  default: '{{.User}}' # from OIDC
  required: true
  disabled: true

- name: dough
  label: Dough kind
  default: "thin"
  options:
    - label: Hand made
      value: hand-made
    - label: Thin crust
      value: thin

- name: cheese
  label: Pick cheese
  required: true
  multiple: true
  options:
    - label: Italian Mozzarella
      value: mozzarella
    - label: Spanish Cheddar
      value: cheddar
    - label: Something Else
      value: something

- name: phone
  label: Phone number
  required: true
  description: Please use real phone number - we will contact you
```


## Types

| Type      | Format             | Example          |
|-----------|--------------------|------------------|
| string    | any string         | foo bar baz      | 
| integer   | number             | 1234             |
| float     | number with dot    | 12345.789        |
| boolean   | `true/false`       | false            |
| date      | `YYYY-MM-DD`       | 2023-01-30       |
| date-time | `YYYY-MM-DDTHH:mm` | 2023-01-30T16:05 |

Notes:

- `boolean` requires [string representation](https://pkg.go.dev/strconv#ParseBool) of true/false
- time/date handling in Go is quite [special](https://pkg.go.dev/time#pkg-constants):
    - use `2006-01-02` for date
    - use `2006-01-02T15:04` for date-time

## Option

| Name      | Type   | Default | Description                                         |
|-----------|--------|---------|-----------------------------------------------------|
| *`label`* | string |         | UI visible label for the option                     |
| `value`   | string |         | Value for the option which will be saved in storage |

```yaml
- name: dough
  label: Dough kind
  default: "thin"
  options:
    - label: Hand made
      value: hand-made
    - label: Thin crust
      value: thin
```

<!-- {% endraw %} -->