# Prefill

Prefill, or default values can be set via `default` section in yaml.

For example, if you want to prefill field from query param:


```yaml
  - name: email
    label: EMail
    default: '{{.Query.Get "email" }}'
```

Then you can pre-fill email like:

    https://my-site/forms/my-form?email=foo@bar.baz

Supports all type except `multiple: true` (arrays).

