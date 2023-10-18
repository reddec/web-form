# Templates

<!--  {% raw %} --> 

For `default` values, for result messages, for description
the [Go template](https://pkg.go.dev/text/template#hdr-Actions) can be used.

In addition to standard library, the following extra additions are avaialable:

- everything from [sprig](http://masterminds.github.io/sprig/)
- `markdown` filter which converts any text to markdown using GFM syntax
- `html` filter which marks any text as safe HTML (danger!)
- `timezone` filter which returns name of current timezone, commonly it's just `Local` though not very useful.

The result of evaluation should match [type](fields.md#types) of field.

## Context for defaults

Context variables accessible as `{{$.<VAR NAME>}}`.
For example `{{$.User}}` will return username or empty string.

> Note: if you advance user of Go templates, then you know when `$` can be omitted,
> otherwise it's safe to use it always.

| Name      | Type                                            | Description                                                                      |
|-----------|-------------------------------------------------|----------------------------------------------------------------------------------|
| `Headers` | [url.Values](https://pkg.go.dev/net/url#Values) | Access to raw request headers                                                    |
| `Query`   | [url.Values](https://pkg.go.dev/net/url#Values) | Access to raw request URL query params                                           |
| `Form`    | [url.Values](https://pkg.go.dev/net/url#Values) | Access to raw request form values                                                |
| `User`    | string                                          | (optional) username from OIDC claims                                             |
| `Groups`  | []string                                        | (optional) list of user groups from OIDC claims                                  |
| `Email`   | string                                          | (optional) user email from OIDC claims                                           |
| `Code`    | string                                          | (optional) [access code](authorization.md#codes) used by user to access the form |

## Context for notifications

| Name     | Type                                            | Description            |
|----------|-------------------------------------------------|------------------------|
| `Form`   | [form definititon](../internal/schema/types.go) | Parsed form definition |
| `Result` | `map[string]any`                                | Result from storage    |

## Context for result

| Name     | Type                                            | Description                  |
|----------|-------------------------------------------------|------------------------------|
| `Form`   | [form definititon](../internal/schema/types.go) | Parsed form definition       |
| `Result` | `map[string]any`                                | Optional result from storage |
| `Error`  | `error`                                         | Optional error               |

If `.Error` is defined, then `.Result` is `nil`.

<!-- {% endraw %} -->