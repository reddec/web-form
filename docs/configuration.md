# Configuration

Here is general server configuration.

- for fields configuration see additional [document](./fields.md)
- for storage configuration see [stores](./stores.md)
- for OIDC configuration see [authorization](./authorization.md)

Be aware that CLI defaults and [docker](docker.md) defaults may be different.

All configuration parameters can be set via environment variables (`$VARNAME`) or via command line arguments. In
documentation most of the time then environment variables will be used for example as a recommended way to configure
application.

There are **no required** fields, default values will let application start normally.

**usage**

```
Application Options:
--configs=                      File or directory with YAML configurations (default: configs) [$CONFIGS]
--storage=[database|files]      Storage type (default: database) [$STORAGE]
--server-url=                   Server public URL. Used for OIDC redirects. If not set - it will try to deduct [$SERVER_URL]
--disable-listing               Disable listing in UI [$DISABLE_LISTING]

Database storage:
--db.dialect=[postgres|sqlite3] SQL dialect (default: sqlite3) [$DB_DIALECT]
--db.url=                       Database URL (default: file://form.sqlite) [$DB_URL]
--db.migrations=                Migrations dir (default: migrations) [$DB_MIGRATIONS]
--db.migrate                    Apply migration on start [$DB_MIGRATE]

Files storage:
--files.path=                   Root dir for form results (default: results) [$FILES_PATH]

Webhooks general configuration:
--webhooks.buffer=              Internal queue size before processing (default: 100) [$WEBHOOKS_BUFFER]

AMQP configuration:
--amqp.url=                     AMQP broker URL (default: amqp://guest:guest@localhost) [$AMQP_URL]
--amqp.buffer=                  Internal queue size before processing (default: 100) [$AMQP_BUFFER]
--amqp.workers=                 Number of parallel publishers (default: 4) [$AMQP_WORKERS]

HTTP server configuration:
--http.bind=                    Binding address (default: :8080) [$HTTP_BIND]
--http.disable-xsrf             Disable XSRF validation. Useful for API [$HTTP_DISABLE_XSRF]
--http.tls                      Enable TLS [$HTTP_TLS]
--http.key=                     Private TLS key (default: server.key) [$HTTP_KEY]
--http.cert=                    Public TLS certificate (default: server.crt) [$HTTP_CERT]
--http.read-timeout=            Read timeout to prevent slow client attack (default: 5s) [$HTTP_READ_TIMEOUT]
--http.write-timeout=           Write timeout to prevent slow consuming clients attack (default: 5s) [$HTTP_WRITE_TIMEOUT]
--http.assets=                  Directory for assets (static) files [$HTTP_ASSETS]

OIDC configuration:
--oidc.enable                   Enable OIDC protection [$OIDC_ENABLE]
--oidc.client-id=               OIDC client ID [$OIDC_CLIENT_ID]
--oidc.client-secret=           OIDC client secret [$OIDC_CLIENT_SECRET]
--oidc.issuer=                  Issuer URL (without .well-known) [$OIDC_ISSUER]
--oidc.redis-url=               Optional Redis URL for sessions. If not set - in-memory will be used [$OIDC_REDIS_URL]
--oidc.redis-idle=              Redis maximum number of idle connections (default: 1) [$OIDC_REDIS_IDLE]
--oidc.redis-max-connections=   Redis maximum number of active connections (default: 10) [$OIDC_REDIS_MAX_CONNECTIONS]

Cloudflare Turnstile:
--captcha.turnstile.site-key=   Widget access key [$CAPTCHA_TURNSTILE_SITE_KEY]
--captcha.turnstile.secret-key= Server side secret key [$CAPTCHA_TURNSTILE_SECRET_KEY]
--captcha.turnstile.timeout=    Validation request timeout (default: 3s) [$CAPTCHA_TURNSTILE_TIMEOUT]
```

- By-default, by the root path `/` listing of all forms available. It can be disabled by `DISABLE_LISTING=true`

## Captcha

*since 0.4.0*

WebForms offers CAPTCHA functionality for POST requests, which serves as a mechanism for verifying the
authenticity of incoming requests during both form submission and access code submission processes. To enable CAPTCHA,
JavaScript on the client side is a prerequisite.

Currently only [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/) captcha is supported.

## HTTP and TLS

Service supports HTTPS but doesn't support dynamic reload. If you are using short-lived certificates such as Let's
Encrypt it could be necessary to restart application after renewal.

**Example configuration**

```
HTTP_TLS=yes
HTTP_KEY=/path/to/key.pem
HTTP_CERT=/path/to/cert.pem
```

### Assets

since 0.2.0

WebForms can handle user-defined static files (assets) via `/assets/` path if `--http.assets` flag is defined.
By-default, it's disabled in CLI mode and enabled in [docker](docker.md) mode.

It could be useful for embedding images or other things in templates. For example:

```yaml
---
table: shop
title: Order Pizza
description: |
  Welcome to our pizzeria!

  ![](/assets/logo.png)

# other configuration
```

### Security

The service has incorporated built-in protection
against [CSRF](https://en.wikipedia.org/wiki/Cross-site_request_forgery) by employing an HTTP-only cookie and a
concealed input (Double Submit Cookie) mechanism. However, if the service is to be utilized programmatically, such as
through an API, CSRF validation may pose potential issues. To disable this validation, you can
set `HTTP_DISABLE_XSRF=true`, but exercise caution and ensure a thorough understanding of the
associated [risks](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
before proceeding.

The service configures protective headers to thwart clickjacking, provide XSS protection, and enforce a strict referer
policy. For now, this protection can not be disabled. You can find the complete list of headers below:

| Header                   | Value                             |
|--------------------------|-----------------------------------|
| `X-Frame-Options`        | `DENY`                            |
| `X-XSS-Protection`       | `1`                               |
| `X-Content-Type-Options` | `nosniff`                         |
| `Referrer-Policy`        | `strict-origin-when-cross-origin` |

## Production checklist

The recommended configuration checklist for the production:

- [ ] put service behind reverse-proxy with TLS (nginx, caddy, haproxy, etc..)
- [ ] secure connection between reverse-proxy and service via internal certificates (overkill for small/medium setups)
- [ ] set public server URL (`--server-url/$SERVER_URL`)
- [ ] use [postgres storage](./stores.md#postgres) with TLS
- [ ] use [OIDC](./authorization.md) with Redis for session storage
- [ ] use non-privileged user for service/container (non-root is supported natively)
- [ ] use specific version of container and reference it by digest
- [ ] set correct permissions for directories:
    - [ ] read only for configs (`/configs` by default for container)
    - [ ] read only for migrations (if applicable) (`/migrations` by default for container)
    - [ ] read only for TLS files
    - [ ] for data directory (`/data` by default for container)
        - write-only for `files`
        - read-write for `sqlite3`
        - no access needed for `postgres`

For extra safety (overkill for small/medium setups) migrations for database can be done outside of the service and
therefore
minimal possible (`INSERT` only for specific tables) permission can be issued.