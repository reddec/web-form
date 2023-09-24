# Web Forms

WebForms is a versatile tool with a focus on DevOps compatibility, designed for creating HTML UI forms
with [backends](./stores.md) that can be hosted in PostgreSQL, SQLite3, or plain JSON files. The service offers the
flexibility to reuse existing tables and database structures and includes automated embedded schema migration.

The user interface (UI) is designed to be lightweight and mobile-friendly, and it can function without JavaScript,
although there may be some limitations in timezone detection. The service also includes various security features.
Additionally, there is an option to disable the UI for browsing available forms. The service also
offers [OIDC](authorization.md) integration for authorization, with the ability to use OIDC claims, such as usernames,
in templating and default values.

WebForms allows for the sending of multiple POST requests ([WebHooks](webhooks.md)) after form submissions to facilitate
integration with external systems. It provides a configurable retry strategy for reliability.

Flexible [templating](template.md) enables the prefilling of fields and the generation of personalized greeting
messages.

Initial setup of the service is straightforward and requires minimal backend configuration and form definition. However,
for those who require more customization, almost every aspect of the service can be [configured](configuration.md). It
can be used in a stateless manner and is scalable. Refer to
the [production checklist](configuration.md#production-checklist) for further details.

WebForms is available in various formats, including [source code](https://github.com/reddec/web-form), pre-compiled
[binaries](https://github.com/reddec/web-form/releases/latest) for major platforms,
and [containers](https://github.com/reddec/web-form/pkgs/container/web-form) for both AMD and ARM
architectures.

The project is licensed under MPL-2.0 (Exhibit A), which allows for commercial usage with minimal restrictions, provided
that any changes made are shared with the community. This promotes collaboration and community involvement.

## Installation

- From source code using go 1.21+ `go install github.com/reddec/web-form/cmd/...@latest`
- From [binaries](https://github.com/reddec/web-form/releases/latest)
- From [containers](https://github.com/reddec/web-form/pkgs/container/web-form) - see [docker](./docker.md)

## Quick start

Let's imagine situation when you are going to collect opinions about which pizzas to order into the office.

Create file `order-pizza.yaml` in directory `configs` with the following content:

```yaml
title: Let's order pizza!
table: pizza
description: |
  Dear colleagues, I'm going to order some pizza this Friday.

  Please, write your preferences bellow.
fields:
  - name: employee
    label: Your name
    required: true
  - name: pizza_kind
    label: Which pizza do you want?
    required: true
    options:
      - label: Hawaii
      - label: 4-cheese
      - label: Meaty
  - name: extra_cheese
    label: Do you want extra cheese?
    type: boolean
    default: "true"
```

Create directory `data`.

Run docker container

    docker run --rm -v $(pwd)/configs:/configs:ro -v $(pwd)/data:/data -p 8080:8080 ghcr.io/reddec/web-form:latest

Open in you browser http://localhost:8080 and click to the form (direct link will
be http://localhost:8080/forms/order-pizza).

Once submitted - check `data` directory. It will contain your result. Of course, the main power comes with
proper [configuration](configuration.md).

![Screenshot 2023-09-24 211012](https://github.com/reddec/web-form/assets/6597086/605400e1-c660-4c95-a59a-ba20ab70d1ed)

## Examples

Check examples in corresponding [directory](https://github.com/reddec/web-form/tree/master/examples).

## Next steps

- Read about [form](form.md) definitions
- And about [configuration](configuration.md)