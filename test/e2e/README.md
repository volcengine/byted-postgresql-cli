# PostgreSQL CLI E2E Tests

This directory contains executable end-to-end tests for
`byted-postgresql-cli`. The existing `test/script` directory remains a
collection of command templates and manual verification scripts; it does not
assert command results.

## Test Modes

### Replay mode

Replay mode is the default and does not use cloud credentials. It builds the
CLI in a temporary directory, runs the binary as a subprocess, and verifies
exit codes, output, JSON payloads, and command help.

```bash
make test-e2e-replay
```

### Live mode

Live mode is opt-in and currently contains read-only workflows. It requires a
dedicated test account and an existing test Workspace and Branch:

```bash
VOLCENGINE_E2E_MODE=live \
VOLCENGINE_E2E_ACCESS_KEY='<access-key-id>' \
VOLCENGINE_E2E_SECRET_KEY='<secret-access-key>' \
VOLCENGINE_REGION='cn-beijing' \
VOLCENGINE_E2E_WORKSPACE_ID='<workspace-id>' \
VOLCENGINE_E2E_BRANCH_ID='<branch-id>' \
make test-e2e-live
```

Live tests do not create or delete cloud resources in this first phase. Do not
use production resource IDs. Credentials are read from the environment and
are never written to fixtures or test output.

### Pre-provisioned Console Login

Live E2E tests can also use a previously prepared Console Login profile. The
profile directory must contain the `.volcengine` directory contents, including
`config.json` and `login/cache`:

```bash
VOLCENGINE_E2E_MODE=live \
VOLCENGINE_E2E_AUTH_MODE=profile \
VOLCENGINE_E2E_PROFILE_DIR="$HOME/.volcengine" \
VOLCENGINE_E2E_PROFILE='e2e-console' \
VOLCENGINE_REGION='cn-beijing' \
VOLCENGINE_E2E_WORKSPACE_ID='<workspace-id>' \
VOLCENGINE_E2E_BRANCH_ID='<branch-id>' \
make test-e2e-live
```

`VOLCENGINE_E2E_PROFILE` defaults to `default`. The harness copies the
specified profile directory into an isolated temporary `HOME`; it does not
modify the source profile or use the developer's default configuration
directly. The cached Console Login session must still be valid or contain a
refresh token.

## Layout

```text
test/e2e/
├── authentication/    configure, login, logout, and status
├── discovery/         projects, workspaces, branches, and operations
├── database_access/   connection, query, dump, pull, and inspect
├── database_resources/ computes, databases, roles, and endpoints
├── workspace_settings/ network and tags
├── tools/             MCP, update, version, and help
├── harness/  subprocess execution, isolation, assertions, and modes
└── README.md
```

## Safety Rules

- Replay tests must not require credentials or network access.
- Live tests must remain explicitly gated by `VOLCENGINE_E2E_MODE=live`.
- The first live phase is read-only.
- Never print AK/SK, STS tokens, passwords, or full connection strings.
- Console Login profile directories are sensitive and must not be committed or
  printed in test output.
- Add resource provisioning and cleanup only with a dedicated cleanup harness
  and a separate CI job.
