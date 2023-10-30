# Notifications

<!--  {% raw %} --> 

WebForm provides seamless integration with your business logic through notifications, which are triggered after form
submission. You can easily integrate almost any self-hosted solution with WebForms. Additionally, cloud-based solutions
like [Workato](https://www.workato.com/) or [IFTTT](https://ifttt.com/) can be integrated using webhooks or direct API
calls.

There is no fixed order for handling notifications. All notifications are dispatched in parallel. However, depending on
the notification sub-system, some messages may be sent in a semi-sequential manner.

By default, all notification sub-systems come with reasonable retry and timeout settings. To avoid blocking form
submissions, all notifications are first enqueued in an internal in-memory queue and then processed by the notification
sub-system. You can configure the size of the queue for each sub-system using the `buffer` parameter. If the queue
becomes full, further submissions will be blocked.

## Webhooks

For each form submission an HTTP sub-request from the server can be performed to any other resource (webhook).

The HTTP `method` is by default `POST`, `headers` has no predefined values. If  `message` is not set, payload is JSON
representation of storage result (effectively newly created object), otherwise it is interpreted
as [template](template.md#context-for-notifications).

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

### Type

| Field      | Type                                              | Default | Description                                                          |
|------------|---------------------------------------------------|---------|----------------------------------------------------------------------|
| **`url`**  | string                                            |         | WebHook HTTP(s) URL. Required                                        |
| `method`   | string                                            | POST    | HTTP method (GET, POST, PUT, etc...)                                 |
| `retry`    | int                                               | 3       | Maximum number of retries                                            |
| `timeout`  | [Duration](https://pkg.go.dev/time#ParseDuration) | 10s     | Request timeout                                                      |
| `interval` | [Duration](https://pkg.go.dev/time#ParseDuration) | 15s     | Interval between retries                                             |
| `headers`  | map[string]string                                 |         | Any additional headers, for example `Authorization`                  |
| `message`  | string                                            |         | [template](template.md#context-for-notifications for message payload |

Notes:

- negative `retry` disables retries
- empty `message` means JSON representation of the result returned by storage

Updates:

- `method` supported since 0.2.0

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

## AMQP

*since 0.3.0*

For a more robust notification solution, consider using a message broker. Currently, only AMQP 0.9.1 brokers are
supported. RabbitMQ is the officially tested option, but other brokers should work seamlessly, as there are no specific
dependencies tied to RabbitMQ.

AMQP notifications typically provide at-least-once delivery guarantees. Therefore, it's advisable to implement
client-side deduplication using attributes like message ID. In practice, duplicate messages may arise only in case of
network delays.

Connections to the brokers are established in a lazy manner. You can configure the maximum number of parallel
submissions using the `workers` parameter (see [configuration](configuration.md#configuration)). WebForms will
automatically reconnect to the broker in case of any issues.
WebForm doesn't handle the definition of AMQP objects such as exchanges, queues, or bindings.
This responsibility lies with the user.

The minimal definition is `key` only:

```yaml
amqp:
  - key: "events.form-submission"
```

### Global configuration

```
AMQP configuration:
--amqp.url=                     AMQP broker URL (default: amqp://guest:guest@localhost) [$AMQP_URL]
--amqp.buffer=                  Internal queue size before processing (default: 100) [$AMQP_BUFFER]
--amqp.workers=                 Number of parallel publishers (default: 4) [$AMQP_WORKERS]
```

### Type

| Field         | Type                                              | Default | Description                                                           |
|---------------|---------------------------------------------------|---------|-----------------------------------------------------------------------|
| **`key`**     | string                                            |         | [template](template.md#context-for-notifications) for routing key     |
| `exchange`    | string                                            |         | exchange name                                                         |
| `type`        | string                                            |         | content type property (see notes)                                     |
| `id`          | string                                            |         | [template](template.md#context-for-notifications) for message ID      |
| `correlation` | string                                            |         | [template](template.md#context-for-notifications) for correlation ID  |
| `retry`       | int                                               | 3       | Maximum number of retries to publish message                          |
| `timeout`     | [Duration](https://pkg.go.dev/time#ParseDuration) | 10s     | Publish timeout                                                       |
| `interval`    | [Duration](https://pkg.go.dev/time#ParseDuration) | 15s     | Interval between retries                                              |
| `headers`     | map[string]string                                 |         | Any additional headers, for example `Authorization`                   |
| `message`     | string                                            |         | [template](template.md#context-for-notifications) for message payload |

- negative `retry` disables retries
- empty `message` means JSON representation of the result returned by storage
- if `type` and `message` are not specified, `type` will be set to `application/json`
- in RabbitMQ, if `exchange` is not set, `key` acts as queue name

> Due to AMQP protocol specification content type in header is not the same as `type` (which is mapped to ContentType)
> property in message.

<!-- {% endraw %} -->