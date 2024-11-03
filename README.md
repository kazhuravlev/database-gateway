# Database Gateway

## Quickstart with example setup

Run commands to get a local dbgw instance with 3 postgres.

```shell
git clone git@github.com:kazhuravlev/database-gateway.git
cd database-gateway/example
docker compose up -d
open 'http://user:password@127.0.0.1:8080'
```

You will see a 3 postgres instances (`local-1`, `local-2`, `local-3`): ![pic1_instances.png](example/pic1_instances.png)

Choose `local-1`, put this query `select id, name from clients` and click `Run` ![pic2_run.png](example/pic2_run.png)

## Interesting projects

- https://github.com/vitessio/vitess
- https://github.com/xwb1989/sqlparser
- https://github.com/cockroachdb/cockroach/pkg/sql/parser
- https://github.com/auxten/postgresql-parser
- https://github.com/topics/sql-parser?l=go
- https://play.openpolicyagent.org/
