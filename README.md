# Web Forms

WebForms is a versatile tool with a focus on DevOps compatibility, designed for creating HTML UI forms
with [backends](https://web-form.reddec.net/stores) that can be hosted in PostgreSQL, SQLite3, or plain JSON files. The service offers the
flexibility to reuse existing tables and database structures and includes automated embedded schema migration.

The user interface (UI) is designed to be lightweight and mobile-friendly, and it can function without JavaScript,
although there may be some limitations in timezone detection. The service also includes various security features.
Additionally, there is an option to disable the UI for browsing available forms. The service also
offers [OIDC](https://web-form.reddec.net/authorization) integration for authorization, with the ability to use OIDC claims, such as usernames,
in templating and default values.

WebForms allows for the sending of multiple notifications (ex: [WebHooks](https://web-form.reddec.net/notifications#webhooks)
or [AMQP](https://web-form.reddec.net/notifications#amqp)) after form submissions to facilitate integration with external systems.
It provides a configurable retry strategy for reliability.

Flexible [templating](https://web-form.reddec.net/template) enables the prefilling of fields and the generation of personalized greeting
messages.

Initial setup of the service is straightforward and requires minimal backend configuration and form definition. However,
for those who require more customization, almost every aspect of the service can be [configured](https://web-form.reddec.net/configuration). It
can be used in a stateless manner and is scalable. Refer to
the [production checklist](https://web-form.reddec.net/configuration#production-checklist) for further details.

WebForms is available in various formats, including [source code](https://github.com/reddec/web-form), pre-compiled
[binaries](https://github.com/reddec/web-form/releases/latest) for major platforms,
and [containers](https://github.com/reddec/web-form/pkgs/container/web-form) for both AMD and ARM
architectures.

The project is licensed under MPL-2.0 (Exhibit A), which allows for commercial usage with minimal restrictions, provided
that any changes made are shared with the community. This promotes collaboration and community involvement.


Read docs for details: https://web-form.reddec.net/

![image](https://github.com/reddec/web-form/assets/6597086/b4dce0e1-30cf-492d-96a4-dbcc98eb787d)

## Installation

- From source code using go 1.21+ `go install github.com/reddec/web-form/cmd/...@latest`
- From [binaries](https://github.com/reddec/web-form/releases/latest)
- From [containers](https://github.com/reddec/web-form/pkgs/container/web-form) - see [docker](https://web-form.reddec.net/docker)


## Examples

Check examples in corresponding [directory](https://github.com/reddec/web-form/tree/master/examples).
