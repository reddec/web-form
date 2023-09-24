# Form

<!--  {% raw %} --> 

Form is yaml document stored under `configs` directory (see [configuration](./configuration.md)).
One file may contain multiple form definitions (multi-document YAML); however, in that case `name` should be
explicitly set since it's inferred from file name (without extension) and must be unique.

Supported extensions: `.yaml`, `.yml`, `.json`.

The only required field is `table` which defines table name for database [storage](stores.md) or directory for `files`
mode.

See [examples](https://github.com/reddec/web-form/tree/master/examples) for inspirations.

| Field         | Type                     | Description                                                                              |
|---------------|--------------------------|------------------------------------------------------------------------------------------|
| `name`        | string                   | unique form name, if not set - file name without extension will be used                  |
| `table`       | string                   | database table name (database mode), or directory name (files mode)                      |
| `title`       | string                   | short form title/name                                                                    |
| `description` | string                   | **markdown + [template](template.md)** description of the form                           |
| `fields`      | [][Field](fields.md)     | list of fields definitions                                                               |
| `webhooks`    | [][Webhook](webhooks.md) | list of webhooks                                                                         |
| `success`     | string                   | **markdown + [template](template.md)** message to show in case submission was successful |
| `failed`      | string                   | **markdown + [template](template.md)** message to show in case submission failed         |


Default message for `success`:

    Thank you for the submission!
  
Default message for `faield`:

    Something went wrong: `{{.Error}}`


**Comprehensive example:**

```yaml
---
table: shop
title: Order Pizza
description: |
  Welcome {{.User}}!
  
  Order HOT pizza RIGHT NOW and get
  **huge** discount!
  
  _T&C_ can be applied
fields:
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

  - name: email
    label: EMail
    pattern: '[^@]+@[^@]+'
    default: "{{.Email}}" # from OIDC
    required: true

  - name: notify_sms
    label: Notify by SMS
    type: boolean

  - name: zip
    label: ZIP code
    required: true
    type: integer

  - name: address
    label: Full address
    required: true
    multiline: true

success: |
  ## Thank you!
  
  Your order {{or .Result.ID .Result.id}} is on the way

failed: |
  ## Sorry!
  
  Something went wrong. Please contact our support and tell them the following message:
  
      {{.Error}}
  

webhooks:
  - url: https://example.com/new-pizza
    name: order
    retry: 3
    interval: 10s
    timeout: 30s
    method: PUT

  - url: https://example.com/notify-to-telegram
    message: |
      #{{ .Result.ID }} New pizza ordered.
```
<!-- {% endraw %} -->