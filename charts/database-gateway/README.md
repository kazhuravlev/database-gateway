# database-gateway Helm chart

This chart deploys Database Gateway as a single stateless web pod backed by:

- one metadata PostgreSQL database for internal storage
- one or more external PostgreSQL targets
- an OIDC provider
- one or more OPA policy files mounted into the container

## Install

Create a values file and provide at least:

- `config.facade.cookieSecret`
- `config.storage.*`
- `config.users.*`
- `config.targets`
- `policy.files`

Example:

```yaml
config:
  facade:
    cookieSecret: "replace-me"
  storage:
    host: "postgres-rw.default.svc"
    port: 5432
    database: "local__dbgw"
    username: "local__dbgw"
    password: "replace-me"
    useSSL: false
    maxPoolSize: 16
  users:
    clientID: "db-gateway"
    clientSecret: "replace-me"
    issuerURL: "https://auth.example.com/application/o/db-gateway/"
    redirectURL: "https://dbgw.example.com/auth/callback"
    accessTokenAudience: "db-gateway"
    scopes:
      - groups
      - email
      - profile
    roleClaim: "groups"
    roleMapping:
      dbgw-admins: admin
      dbgw-users: user
  targets:
    - id: "local-1"
      description: "Production for clients service"
      tags:
        - "env:production"
        - "svc:clients"
      type: "postgres"
      connection:
        host: "postgres1-rw.default.svc"
        port: 5432
        user: "pg01"
        password: "replace-me"
        db: "pg01"
        use_ssl: false
        max_pool_size: 4
      default_schema: "public"
      tables:
        - table: "public.clients"
          fields:
            - "id"
            - "name"
            - "email"

policy:
  files:
    gateway.rego: |
      package gateway

      default allow_target := false
      default allow_query := false

      allow_target if {
        "role:admin" in input.subjects
      }

      allow_query if {
        "role:admin" in input.subjects
      }

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: dbgw.example.com
      paths:
        - path: /
          pathType: Prefix
```

Install with:

```bash
helm upgrade --install dbgw ./charts/database-gateway -f values.yaml
```

OCI install from GHCR after a tagged release:

```bash
helm pull oci://ghcr.io/kazhuravlev/charts/database-gateway --version <chart-version>
helm install dbgw oci://ghcr.io/kazhuravlev/charts/database-gateway --version <chart-version> -f values.yaml
```

## Notes

- The chart stores `config.json` in a Kubernetes `Secret` because it contains credentials.
- The application has no dedicated health endpoint today, so the chart uses `tcpSocket` probes.
- Migrations run automatically during startup because the binary already executes `migrate-up` before serving traffic.
