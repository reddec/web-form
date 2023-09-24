# Webhooks
<!--  {% raw %} --> 

For each form submission an HTTP sub-request from the server can be performed to any other resource (webhook).

The HTTP `method` is by default `POST`, `headers` has no predefined values. If  `message` is not set, payload is JSON
representation of storage result (effectively newly created object), otherwise it is interpreted
as [template](template.md#context-for-webhooks).

There is no limits for reasonable number of webhooks and number is limited only by server resources (CPU mostly).

The server will retry delivery up to `retry` times (default is 3), with constant `interval` between attempts (default is
10 seconds) until remote resource will return 2xx (200, 201, ..., 299) code. A webhook request duration is limited
to `timeout` per attempt (default 30 seconds).

Delivery made in non-blocking semi-parallel way, after saving information to the storage, and only in case of success.

The minimal definition is `url` only:

```yaml
webhooks:
  - url: https://example.com/new-pizza
```

The full definition is:

```yaml
webhooks:
  - url: https://example.com/new-pizza
    retry: 3
    interval: 10s
    timeout: 30s
    method: POST
    message: |
      New pizza order #{{ .Result.ID }}.
```

<!-- {% endraw %} -->