# Authorization

By-default, the solution has no authorization. However, it is possible to use
any [OIDC](https://auth0.com/docs/authenticate/protocols/openid-connect-protocol#:~:text=OpenID%20Connect%20(OIDC)%20is%20an,obtain%20basic%20user%20profile%20information)
-compliant provider to secure
access to the forms.

OAuth callback URL is `<server-url>/oauth2/callback`. For example, if your public server url
is `https://forms.example.com`, then callback url is `https://forms.example.com/oauth2/callback`.

It uses [oidc-login](https://github.com/reddec/oidc-login) library in order to provide access and can be configured via:

    OIDC configuration:
    --oidc.enable                   Enable OIDC protection [$OIDC_ENABLE]
    --oidc.client-id=               OIDC client ID [$OIDC_CLIENT_ID]
    --oidc.client-secret=           OIDC client secret [$OIDC_CLIENT_SECRET]
    --oidc.issuer=                  Issuer URL (without .well-known) [$OIDC_ISSUER]
    --oidc.redis-url=               Optional Redis URL for sessions. If not set - in-memory will be used [$OIDC_REDIS_URL]
    --oidc.redis-idle=              Redis maximum number of idle connections (default: 1) [$OIDC_REDIS_IDLE]
    --oidc.redis-max-connections=   Redis maximum number of active connections (default: 10) [$OIDC_REDIS_MAX_CONNECTIONS]

    Application Options:
    --server-url=                   Server public URL. Used for OIDC redirects. If not set - it will try to deduct [$SERVER_URL]

Required:

- `oidc.enable` should be set to `true`
- `oidc.client-id`, `oidc.client-secret` should be set to credentials from OIDC provider (also known as "private" mode)
- `oidc.issuer` url for the issuer without `.well-known` path.

Recommended:

- `server-url` to avoid any problems with callback urls
- `oidc.redis-url` to share sessions between instances

**Example configuration:**

```
OIDC_ENABLE=true
OIDC_CLIENT_ID=my-super-forms
OIDC_CLIENT_SECRET=M0QmHyHUqt1z17L8XwYiQeah9l1U7zNh
OIDC_ISSUER=https://auth.example.com/realms/reddec
```