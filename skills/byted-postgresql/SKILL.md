---
name: byted-postgresql
description: 火山引擎 PostgreSQL CLI. 支持管理 AIDAP PostgresSQL 项目、工作空间、分支、计算资源、数据库、角色、终端节点、网络与标签管理，并提供 SQL 查询、Schema Diff、MCP、psql、pg_dump 和数据库诊断功能。
---

# Byted PostgreSQL

Use the locally packaged `byted-postgresql-cli` binary for Volcengine PostgreSQL control-plane and data-plane workflows.

## Region and Credentials

- Volcengine credentials use `~/.volcengine` and `VOLCENGINE_*`.
- The default region is `cn-beijing`.
- Use explicit regions for automation.

Prefer explicit regions for automation:

```bash
byted-postgresql-cli --region cn-beijing status
```

## Read-Only Inspection

Check `status` first, pass workspace and branch IDs explicitly, and prefer JSON output. Do not run create, delete, start, stop, rename, configure, or database write commands unless explicitly requested.

```bash
byted-postgresql-cli --region <region> workspaces get <workspace-id> --output json
byted-postgresql-cli --region <region> branches get <branch-id> --workspace-id <workspace-id> --output json
byted-postgresql-cli --region <region> computes list --workspace-id <workspace-id> --branch-id <branch-id> --output json
byted-postgresql-cli --region <region> databases list --workspace-id <workspace-id> --branch-id <branch-id> --output json
byted-postgresql-cli --region <region> roles list --workspace-id <workspace-id> --branch-id <branch-id> --output json
```

Respect `total`, `limit`, and `offset` in paginated responses.

## MCP

Use the same provider context for MCP:

```bash
byted-postgresql-cli --region <region> mcp serve \
  --workspace-id <workspace-id> \
  --read-only
```

MCP registers the Volcengine PostgreSQL resource tools over stdio.

## Database Access

Use `connection-string` or `psql` only when data-plane access is explicitly requested. Never expose passwords, STS tokens, or connection URLs in logs or generated artifacts.

See `references/volcengine.md` for Volcengine authentication, local
data directories, and supported workflows.
