# OPA policy examples for Database Gateway

This directory contains two things:

- `basic/`: the runnable example policy bundle used by `example/config.json`
- numbered `.rego` files: standalone real-world policy examples

## What the current implementation supports

The OPA authorizer in `internal/policy/opa` evaluates only this input:

```json
{
  "subjects": ["user:alice@example.com", "role:user"],
  "target": "taxi-prod",
  "op": "select",
  "table": "public.clients"
}
```
