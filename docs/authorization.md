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

## Access control

since: 0.2.0

Each form has optional field `policy` which is a [CEL](https://github.com/google/cel-spec/blob/master/doc/intro.md)
expression which can be used to define policy of who can access the form.

- If OIDC is not set there are no restrictions (`policy` effectively is ignored)
- If `policy` absent, it is ignored - all authorized users has access to the form
- If `policy` returns `false` or non-convertable to boolean value - users will not be allow access the form

The restriction is also applied for listing - users will list of only allowed forms.

Allowed variables in CEL expression:

- `user` (string) user name
- `groups` ([]string) slice of user's groups, can be empty/nil
- `email` (string) user email, can be empty

`user` is picked from the claims by the following priority (first non-empty):

1. `preferred_username`
2. `email`
3. `sub` (subject)

### Examples

**Group-based access**:

Allow access only for `admin` group

```yaml
policy: '"admin" in groups'
```


Allow access only for `admin` or `sysadmin` groups

```yaml
policy: '"admin" in groups || "sysadmin" in groups'
```

**Domain-based access**:

Allow access only for users with email domain `reddec.net`

```yaml
policy: 'email.endsWith("@reddec.net")'
```

**Per-user access**:

Allow access only for user `reddec`

```yaml
policy: 'user == "reddec"'
```

Allow access only for user `reddec` or `admin`

```yaml
policy: 'user == "reddec" || user == "admin"'
```