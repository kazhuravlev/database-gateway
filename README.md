# Database Gateway

[![Go Reference](https://pkg.go.dev/badge/github.com/kazhuravlev/database-gateway.svg)](https://pkg.go.dev/github.com/kazhuravlev/database-gateway)
[![License](https://img.shields.io/github/license/kazhuravlev/database-gateway?color=blue)](https://github.com/kazhuravlev/database-gateway/blob/master/LICENSE)
[![Test Status](https://github.com/kazhuravlev/database-gateway/actions/workflows/test.yml/badge.svg)](https://github.com/kazhuravlev/database-gateway/actions/workflows/test.yml)
[![Release Status](https://github.com/kazhuravlev/database-gateway/actions/workflows/release.yml/badge.svg)](https://github.com/kazhuravlev/database-gateway/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kazhuravlev/database-gateway)](https://goreportcard.com/report/github.com/kazhuravlev/database-gateway)
[![codecov](https://codecov.io/gh/kazhuravlev/database-gateway/graph/badge.svg?token=DLOML3FTN1)](https://codecov.io/gh/kazhuravlev/database-gateway)

This service provides a unified web interface for secure, controlled access to company databases. It enables employees
to run queries on `production` databases while enforcing access control (`ACL`) policies. For example, team leads may
have permissions to execute both `SELECT` and `INSERT` queries on certain tables, while other team members are
restricted to read-only (`SELECT`) access. This approach ensures that database interactions are managed safely and
that each user's access is tailored to their role and responsibilities.

## Architecture Overview

This application acts as a secure gateway to multiple PostgreSQL instances, allowing authenticated users to run approved
queries through a unified web interface, with fine-grained ACLs controlling access.


```
                     ┌───────────────────────────┐               
                     │ PROD      ┌─────────────┐ │               
                     │       ┌───┤  Postgres1  │ │               
  ┌────────┐    ┌────────┐   │   └─────────────┘ │               
  │  USER  │────│  DBGW  │───┼                   │               
  └────────┘    └────────┘   │   ┌─────────────┐ │               
                     │       └───┤  Postgres2  │ │               
                     │           └─────────────┘ │               
                     └───────────────────────────┘ 
```

### Components

1. **Local PostgreSQL Database**:
    - Stores query results, user profiles, and ACLs.
    - Acts as a cache for query results, allowing unique links for debugging without re-execution.

2. **Remote PostgreSQL Instances**:
    - Host production data and are accessed only through the app.
    - Queries are run only if authorized by ACLs, limiting access to specific users, tables, and query types.

3. **OIDC Authentication**:
    - Users authenticate via an external OIDC provider.
    - User roles are mapped to ACLs, defining what queries each user can run.

4. **Access Control Lists (ACLs)**:
    - Define user permissions at the instance, table, and query type levels.
    - Stored in the local database, restricting queries based on user identity.

5. **Web Interface**:
    - Provides login, query submission, and result viewing.
    - Shows error feedback for unauthorized or restricted queries.

### Flow of Operations

1. **Authentication**: Users log in via OIDC, and their identity maps to ACL permissions.
2. **Query Submission**: Authorized queries are checked against ACLs, then run on remote instances.
3. **Result Caching**: Results are stored locally with unique links for easy access and debugging.

This architecture ensures secure, controlled access to production data, balancing usability with data protection.

## Quickstart with example setup

Run commands to get a local dbgw instance with 3 postgres.

```shell
git clone https://github.com/kazhuravlev/database-gateway.git
cd database-gateway/example
docker compose up --pull always --force-recreate -d
open 'http://127.0.0.1:8080'
# Admin: admin@example.com password
# User1: user1@example.com password
```

You will see only 2 instances from 3 postgres instances (`local-1`, `local-2`,`local-3`) because ACL is applied to test
user. ACLs stored in [config.json](example/config.json).

![pic1_instances.png](example/pic1_instances.png)

Choose `local-1`, put this query `select id, name from clients` and click `Run` ![pic2_run.png](example/pic2_run.png)

## Features

- [x] Supports any PostgreSQL wire-protocol database
- [x] Allows hardcoded user configuration via config file
- [x] Integrates with OpenID Connect for user authentication
- [x] Enforces access filtering through ACLs
- [x] Provides query result output in HTML format
- [x] Provides query result output in JSON format
- [x] Unique links for query results (useful for debugging)
- [x] Fine-grained table-level permissions
- [x] Column-level access control
- [x] Connection pooling for performance optimization
- [x] Query validation and sanitization
- [x] SQL parsing to enforce query type restrictions (SELECT, INSERT, etc.)
- [x] Interactive web UI with keyboard shortcuts (Shift+Enter to run queries)
- [x] Session management with token expiration
- [x] Secure cookie handling

## Advanced Configuration

### Authentication Options

The service supports two authentication methods:

1. **Config File Authentication**: For simple setups with hardcoded users
   ```json
   "users": {
     "provider": "config",
     "configuration": [
       {
         "id": "admin@example.com",
         "username": "admin@example.com",
         "password": "password"
       }
     ]
   }
   ```

2. **OIDC Authentication**: For integration with identity providers
   ```json
   "users": {
     "provider": "oidc",
     "configuration": {
       "client_id": "example-app",
       "client_secret": "example-app-secret",
       "issuer_url": "http://localhost:5556",
       "redirect_url": "http://localhost:8080/auth/callback",
       "scopes": ["email", "profile"]
     }
   }
   ```

### Access Control Configuration

Access control lists define user permissions with fine-grained control:

```json
"acls": [
  {
    "user": "admin@example.com",
    "op": "*",
    "target": "*",
    "tbl": "*",
    "allow": true
  },
  {
    "user": "user1@example.com",
    "op": "select",
    "target": "pg-5433",
    "tbl": "*",
    "allow": true
  }
]
```

Wildcards (`*`) allow all operations, targets, or tables. Specific permissions override broader ones.

### Database Connection Settings

Configure performance settings for each database connection:

```json
"connection": {
  "host": "postgres1",
  "port": 5432,
  "user": "pg01",
  "password": "pg01",
  "db": "pg01",
  "use_ssl": false,
  "max_pool_size": 4
}
```

## Performance Optimizations

- **Connection Pooling**: Configurable connection pool sizes for each database target
- **Query Result Caching**: Results are stored in the local database for later reference
- **Efficient Query Execution**: Parsed and validated for optimal performance

## Security Considerations

- **SQL Injection Protection**: All queries are parsed and validated before execution
- **No Direct Database Access**: Remote databases are only accessible through the gateway
- **Column-Level Restrictions**: ACLs can limit which fields users can query
- **Query Type Restrictions**: Limit users to specific operations (SELECT, INSERT, etc.)
- **Session Security**: Secure cookie handling with configurable expiration
- **Error Handling**: Error messages are sanitized to prevent information leakage

## Edge Cases and Troubleshooting

- **Multiple Schema Support**: Tables can be specified with schema names (`schema.table`)
- **Complex Query Handling**: Some complex queries might be rejected by the parser
- **Connection Failures**: The service gracefully handles database connection failures
- **Authentication Edge Cases**: Fallback strategies when OIDC provider is unavailable
- **Missing Tables/Fields**: Queries referencing unknown tables or fields are rejected
- **ACL Conflicts**: When multiple ACL rules apply, the most specific rule takes precedence

## Interesting projects

- https://github.com/antlr/grammars-v4/tree/master/sql/postgresql/Go
- https://github.com/auxten/postgresql-parser
- https://github.com/blastrain/vitess-sqlparser
- https://github.com/cockroachdb/cockroach/pkg/sql/parser
- https://github.com/pganalyze/pg_query_go/
- https://github.com/pingcap/tidb/tree/master/pkg/parser
- https://github.com/topics/sql-parser?l=go
- https://github.com/vitessio/vitess
- https://github.com/xwb1989/sqlparser
