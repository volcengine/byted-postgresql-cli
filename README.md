# Volcengine PostgreSQL CLI

English | [中文 README](README.zh_CN.md)

`byted-postgresql-cli` is a command-line client for managing PostgreSQL
workspaces on Volcengine. It provides workspace, branch, compute, database,
role, endpoint, operation, schema-diff, PostgreSQL data-plane, and MCP
workflows.

This repository contains the Volcengine PostgreSQL CLI only.

## Requirements

- Node.js and npm for installing the CLI
- A Volcengine account with access to AIDAP PostgreSQL APIs
- `pg_dump` for `db dump` and `db pull`
- `psql` for `psql` and PostgreSQL data-plane commands that use it

## Install

Install the CLI from npm:

```bash
npx @byted-postgresql/cli@latest install 
byted-postgresql-cli --help
```

## Development

To build from source and run the repository test suite:

```bash
make build
byted-postgresql-cli --help
make test
make vet
npm run test:npm
```

Build binaries for the supported release targets:

```bash
make release
```

## Authentication

The CLI supports browser-based Volcengine Console Login and static AK/SK
profiles. Use browser login for normal interactive use:

```bash
byted-postgresql-cli login --region cn-beijing
```

Logout removes local Console Login profiles and cached temporary credentials:

```bash
byted-postgresql-cli logout
```

For automation or static credentials, configure a profile:

```bash
byted-postgresql-cli configure set \
  --profile default \
  --access-key '<access-key-id>' \
  --secret-key '<secret-access-key>' \
  --region cn-beijing
```

An STS session token can be supplied when using temporary credentials:

```bash
byted-postgresql-cli configure set \
  --profile ci \
  --access-key '<access-key-id>' \
  --secret-key '<secret-access-key>' \
  --session-token '<session-token>' \
  --region cn-beijing
```

For CI, credentials can be supplied through environment variables instead of
writing a profile:

```bash
export VOLCENGINE_ACCESS_KEY='<access-key-id>'
export VOLCENGINE_SECRET_KEY='<secret-access-key>'
export VOLCENGINE_SESSION_TOKEN='<session-token>' # optional
export VOLCENGINE_REGION='cn-beijing'              # optional
```

The short environment variable names `VOLC_REGION`, `VOLC_ENDPOINT`, and
`VOLC_PROFILE` are also supported. Long names are available for region and
endpoint through `VOLCENGINE_REGION` and `VOLCENGINE_ENDPOINT`.

Resolution order is:

1. Explicit `--region` or `--profile`
2. Environment variables
3. The selected local profile
4. The default region `cn-beijing`

Do not commit credentials, put them in shell scripts, or print them in CI
logs.

## Local Data

The CLI reuses the Volcengine credential profile format:

```text
~/.volcengine/config.json
```

Browser Console Login caches temporary STS credentials under:

```text
~/.volcengine/login/cache/
```

The CLI stores installation state, update state, and related local data under:

```text
~/.volcengine-postgresql/
```

The `--config-dir` flag controls the CLI's local configuration directory. It
does not change the shared Volcengine credential file. Protect these
directories with normal user-only filesystem permissions.

## Common Commands

Use explicit resource IDs in automation:

```bash
CLI=byted-postgresql-cli

$CLI --region cn-beijing workspaces list
$CLI --region cn-beijing workspaces get <workspace-id>
$CLI --region cn-beijing branches list --workspace-id <workspace-id>
$CLI --region cn-beijing computes list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id>
$CLI --region cn-beijing databases list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id>
$CLI --region cn-beijing roles list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id>
```

List commands use pagination. The default page is limited to 10 items; use
`--offset` and `--limit` to fetch additional pages.

Get a PostgreSQL connection string:

```bash
$CLI --region cn-beijing connection-string \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

The connection string is password-masked by default. Use `--reveal` only when
the password must be printed, and never save the output in logs or shell
history.

## PostgreSQL Data Plane

Preview a dump command without executing `pg_dump`:

```bash
$CLI --region cn-beijing db dump --dry-run \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

The output is a shell-compatible command template. Its password is shown as
`*****`; replace it manually before executing the command.

To preview a complete command with the real password, use `--reveal`. This
prints sensitive data to stdout:

```bash
$CLI --region cn-beijing db dump --dry-run --reveal \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

To execute a dump directly, omit `--dry-run` and write the output to a file:

```bash
$CLI --region cn-beijing db dump \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name> \
  --file /tmp/aidb.sql
```

The `--reveal` and `--masked` flags are only valid with `--dry-run`.

## MCP

Start the MCP server over stdio:

```bash
$CLI --region cn-beijing mcp serve \
  --workspace-id <workspace-id> \
  --read-only
```

Use `--read-only` when integrating with an assistant that should only inspect
resources. MCP responses can contain resource metadata; do not expose
credentials or connection URLs through tool output.

## Contributions

This project is maintained by the Volcengine PostgreSQL team. External
contributions are not currently accepted. Before submitting an internal
change, run:

```bash
gofmt -w .
go test ./...
go vet ./...
npm run test:npm
git diff --check
```

Please keep provider-specific behavior Volcengine-only in this repository and
avoid adding credentials or real workspace identifiers to tests, logs, or
documentation.

## Code of Conduct

Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Security

Please read [SECURITY.md](SECURITY.md) before reporting a security issue.

## License

This project is licensed under the [MIT License](LICENSE).
