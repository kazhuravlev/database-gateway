# SQL authorization contract

The gateway accepts a deliberately restricted PostgreSQL SQL subset. A successful
parse is not permission to execute. The execution boundary accepts only an
`AuthorizedQuery` constructed by `validator.Authorize`; its zero value is rejected.
The original request remains in query history, but the executable SQL comes from
the checked AST.

## Invariants

| Invariant | Enforcement | Regression coverage |
| --- | --- | --- |
| Exactly one statement, no byte truncation or alternate encoding | UTF-8/NUL checks before the native scanner, one `RawStmt`, fixed connection encoding and string-literal mode | `TestRejectUnaccountedSQL`, PostgreSQL integration |
| Bounded input and traversal | 64 KiB SQL, 4096 scanner tokens, 64 SQL nesting levels, 128 AST levels, 16384 AST messages | input/depth tests and fuzzing |
| No silently ignored AST features | Allowlist of populated protobuf fields; every child is visited; unknown messages, fields, wire data and enum numbers fail closed | mutation of every unlisted field; descriptor digest |
| Table identities cannot be replaced by aliases | Independent relation inventory and direct, non-recursive name bindings; duplicate bindings rejected | alias collision/cycle cases, including a denied joined table |
| Policy names are unambiguous | Reject catalog-qualified names, dotted identifier components and column alias lists; aliases never change physical identities | quoted identifiers and alias-list attacks |
| Every referenced relation and column is accounted for | Independent walk of raw `RangeVar` and `ColumnRef` nodes compared with extracted requirements | deliberately omitted dependencies; property tests |
| Expression roles are explicit | Separate INSERT-column, assignment, conflict-index and RETURNING handlers; output aliases never replace input expressions | aliased RETURNING; hidden expressions in upserts |
| Mutations cannot bypass read/update policies | Upserts emit INSERT and UPDATE requirements; DML reads and RETURNING emit SELECT requirements; every requirement must pass | deny each required operation independently |
| No positional field guessing | INSERT requires an explicit column list; unsupported whole-row references and wildcards are rejected (`count(*)` remains supported) | implicit/default INSERT and `count(table.*)` cases |
| Name resolution matches authorization | Qualify relations with the configured default schema and allowlisted functions with `pg_catalog`; execute with `search_path=pg_catalog, pg_temp` | real table-shadowing and function-overload fixtures |
| AST construction preserves semantics | Deparse, reparse, compare all vectors AND the complete protobuf AST except source positions | every supported SQL fixture, round-trip fuzzing |
| No vacuous authorization | Empty requirement sets, missing schema/callback, invalid operations and zero-value authorized SQL fail closed | empty-input and empty-requirement tests |

The AST descriptor digest covers the upstream schema, including currently unused
nodes and enum definitions. Updating `pg_query_go` must trigger review if this
schema changes; do not blindly replace the expected digest. The runtime allowlist
also rejects newly populated fields independently of that test.

The additional coverage walk is intentionally separate from statement handlers.
A handler that forgets a nested column or relation cannot return a successful
analysis. This is a defense against extractor omissions, not an independent
implementation of PostgreSQL's semantic analyzer.

## Problems corrected

- `RETURNING secret AS id` previously recorded `id`. An aliased subquery or function
  could also evade expression checking.
- `ON CONFLICT DO UPDATE SET ...` previously recorded assignment names without
  analyzing their values; hidden subqueries/functions and reads were missed. The
  update was also authorized as INSERT alone.
- Recursive alias mappings could overwrite a physical source, attribute columns
  to another table, hide a joined table, or recurse indefinitely.
- Table column-alias lists could make an allowed name refer to another physical
  column. Flattening quoted dotted identifiers could conflate distinct relations.
- Qualified whole-row function arguments were skipped, positional INSERT did not
  identify written columns, and quoted function names were case-folded incorrectly.
- ACL normalization of unqualified tables did not bind PostgreSQL execution to the
  same schema. Session defaults could change table/function resolution.
- Empty vectors were accepted by public validation helpers.

## Compatibility changes

INSERT must name its columns; `INSERT ... DEFAULT VALUES` is rejected. Explicit
`DEFAULT` values for named columns remain supported. `count(*)` is supported, but
`count(alias.*)` and other whole-row expressions are not. Column-alias lists,
catalog qualifiers and dots inside identifier components are rejected because the
policy format cannot represent them unambiguously. Duplicate/ambiguous table
bindings are rejected conservatively; use distinct aliases for joins.

Upserts now require UPDATE as well as INSERT. RETURNING, mutation filters and
assignment reads require SELECT. Policies that intentionally grant only a write
operation may need review; grant only the operations the caller should have.
Conflict assignments using `excluded.column` remain unsupported; they fail closed
rather than being treated as a different physical table.

Supported SQL remains the explicit corpus in `supported_queries_test.go`. CTEs,
subqueries, set operations, SELECT INTO, locking, casts, arbitrary functions,
window clauses and other unimplemented features are rejected. This change does
not attempt to make arbitrary PostgreSQL SQL safe by enumerating dangerous words.

## Trust boundary and remaining risks

These checks cover explicit SQL dependencies on **configured logical relations**.
They do not resolve database-side object graphs. Views, inherited/partitioned
relations, triggers, rules, RLS policies, defaults, generated columns, foreign keys,
user-defined types, casts, operators and index support functions can read or modify
other objects without adding a raw SQL table/column reference. Configuring a
relation trusts those database-side effects. PostgreSQL inheritance is unchanged;
this PR does not silently rewrite queries to `ONLY`.

The configured schema/field inventory must match the real database, and schemas,
object definitions and the database account must be controlled by trusted
operators. The gateway account must not be a superuser or database owner and must
have only the database privileges it needs. The SQL filter is not a substitute
for PostgreSQL grants or an isolation boundary against a malicious database
administrator. Permissions and schema changes require review even when gateway
code does not change.

Input budgets reduce parser resource exposure. They do not provide a hard CPU or
memory isolation boundary around the C parser, and do not bound database query
runtime, result size, locks, or join cardinality. Configure database-side query
and lock timeouts and resource limits for the gateway role. Fuzzing and finite
regression tests cannot prove the absence of all implementation or upstream bugs.
A hard native-parser deadline requires process isolation, which is outside this
in-process gateway design.

## Verification and extension workflow

CI runs normal/race tests against PostgreSQL 16.15 and two bounded fuzz campaigns.
The real PostgreSQL fixture creates uniquely named schemas, proves that hostile
name resolution and column renaming are possible, and checks the actual gateway
connection and execution path. Without `DBGW_TEST_POSTGRES_DSN`, that integration
test is explicitly skipped. No configured application target is used by the test.

For local checks against a disposable PostgreSQL instance:

```sh
DBGW_TEST_POSTGRES_DSN='postgres://postgres:integration@127.0.0.1:5432/dbgw_test?sslmode=disable' go test -race -timeout 5m ./...
go test ./internal/parser -run '^$' -fuzz '^FuzzParseAndCompile$' -fuzztime 60s -fuzzminimizetime 1s -parallel 2
go test ./internal/validator -run '^$' -fuzz '^FuzzAuthorizationCannotDropColumns$' -fuzztime 60s -fuzzminimizetime 1s -parallel 2
task ci:lint
```

When adding syntax, inspect its actual AST, classify every field, define its
read/write and binding semantics, add allowed and denied examples, and test each
required permission independently. Extend the allowlist only with that support.
Run the round-trip, mutation, fuzz and real-server checks. The embedded parser
currently uses PostgreSQL 17 grammar; the PostgreSQL 16 subset is checked
by integration tests. Other server versions require their own compatibility run.

Primary references: [PostgreSQL SELECT and alias/inheritance semantics](https://www.postgresql.org/docs/16/sql-select.html),
[function resolution and overload hazards](https://www.postgresql.org/docs/16/typeconv-func.html),
[schemas and search paths](https://www.postgresql.org/docs/16/ddl-schemas.html),
and [the upstream PostgreSQL parser/deparser](https://github.com/pganalyze/pg_query_go).
