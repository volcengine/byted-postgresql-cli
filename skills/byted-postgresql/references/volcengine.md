# Volcengine PostgreSQL Capabilities

- Provider: `volcengine`
- Default region: `cn-beijing`
- Console: `https://signin.volcengine.com`
- Config: `~/.volcengine/config.json`
- Environment: `VOLCENGINE_*`
- CLI data root: `~/.volcengine-postgresql/`
- Console Login cache: `~/.volcengine/login/cache/`

The CLI is dedicated to Volcengine PostgreSQL. It does not select or manage
another cloud provider.

## Authentication

Interactive browser login:

```bash
byted-postgresql-cli login --region cn-beijing
```

Static credentials:

```bash
byted-postgresql-cli configure set \
  --access-key '<access-key-id>' \
  --secret-key '<secret-access-key>' \
  --region cn-beijing
```

For CI, use `VOLCENGINE_ACCESS_KEY`, `VOLCENGINE_SECRET_KEY`, and optionally
`VOLCENGINE_SESSION_TOKEN`.

## Resource Workflows

The command tree covers Volcengine PostgreSQL workspaces, branches, computes,
databases, roles, endpoints, operations, schema differences, database
inspection, `pg_dump`, `psql`, and MCP.

Use `--workspace-id` and `--branch-id` explicitly in non-interactive
automation. List commands return paginated results; use `--limit` and
`--offset` to fetch additional pages.

## Sensitive Output

Connection strings and dump previews mask passwords by default. Use
`--reveal` only for an intentional local operation, and never persist its
output to logs, shell history, or generated artifacts.
