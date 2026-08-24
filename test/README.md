# byted-postgresql-cli-volcengine 命令集合脚本

> 面向 `byted-postgresql-cli`（即 `byted-postgresql-cli` 编译产物）的**命令模板脚本合集**。
> 只负责**收集与组织命令**，不承担任何测试判定；破坏性命令**默认注释掉**，可作为日常查阅、CI 参考、故障演练的入口。

---

## 1. 项目背景

火山引擎 Serverless PostgreSQL CLI。


---

## 2. 目录结构

```
script/
├── authentication/
│   ├── configure.sh
│   ├── login.sh
│   ├── logout.sh
│   └── status.sh
├── discovery/
│   ├── branches.sh
│   ├── operations.sh
│   ├── projects.sh
│   └── workspaces.sh
├── database-access/
│   ├── connection-string.sh
│   ├── db.sh
│   ├── inspect.sh
│   ├── psql.sh
│   └── schema-diff.sh
├── database-resources/
│   ├── computes.sh
│   ├── databases.sh
│   ├── endpoints.sh
│   └── roles.sh
├── workspace-settings/
│   ├── network.sh
│   └── tags.sh
└── tools/
    ├── completion.sh
    ├── config.sh
    ├── help.sh
    ├── mcp.sh
    ├── update.sh
    └── version.sh
```

- **第一层**固定为 `script/`。
- **第二层**目录名严格对应 `byted-postgresql-cli --help` 里的 command group。
- **第三层**每个 `.sh` 文件对应一个一级命令，内部按 help 顺序枚举其全部子命令模板。

---


## 4. 通用约定

### 4.1 脚本头部

```bash
#!/usr/bin/env bash
set -euo pipefail

BYTEDCLI_BIN="${BYTEDCLI_BIN:-byted-postgresql-cli}"
TEST_PREFIX="${TEST_PREFIX:-byted-postgresql-cli-test-}"

# Required / Optional environment variables:
# - VOLCENGINE_WORKSPACE_ID
# - VOLCENGINE_BRANCH_ID
# - ...
```

### 4.2 环境变量总览

| 变量名 | 必填 | 说明 |
|---|---|---|
| `BYTEDCLI_BIN` | 否 | CLI 二进制路径或命令名，默认 `byted-postgresql-cli`；使用仓库编译产物时设为 `./bin/byted-postgresql-cli` |
| `TEST_PREFIX` | 否 | 生成测试资源时的统一命名前缀，默认 `byted-postgresql-cli-test-` |
| `BYTEDCLI_PROFILE` | 否 | 目标 profile 名称 |
| `BYTEDCLI_ACCESS_KEY_ID` | 否 | 使用 AK/SK 登录时的 AK |
| `BYTEDCLI_SECRET_ACCESS_KEY` | 否 | 使用 AK/SK 登录时的 SK |
| `BYTEDCLI_REGION` | 否 | 目标地域，默认 `cn-beijing` |
| `VOLCENGINE_WORKSPACE_ID` | 高频 | 目标 workspace ID |
| `VOLCENGINE_BRANCH_ID` | 高频 | 目标分支 ID |
| `VOLCENGINE_PARENT_BRANCH_ID` | 否 | branches children / create 时使用 |
| `VOLCENGINE_SOURCE_BRANCH_ID` / `VOLCENGINE_TARGET_BRANCH_ID` | 否 | `branches diff` 使用 |
| `VOLCENGINE_SOURCE_DATABASE_NAME` / `VOLCENGINE_TARGET_DATABASE_NAME` | 否 | `branches diff` 使用 |
| `VOLCENGINE_SOURCE_SCHEMA_NAME` / `VOLCENGINE_TARGET_SCHEMA_NAME` | 否 | `branches diff` 使用 |
| `VOLCENGINE_WORKSPACE_NAME` | 否 | 创建 / 改名 workspace 使用 |
| `VOLCENGINE_DATABASE_NAME` | 否 | 目标数据库名，默认 `postgres` |
| `VOLCENGINE_ROLE_NAME` | 否 | 目标角色名，默认 `postgres` |
| `VOLCENGINE_ROLE_PASSWORD` | 否 | 角色密码，禁止写死；建议 shell 注入 |
| `VOLCENGINE_COMPUTE_ID` / `VOLCENGINE_COMPUTE_NAME` | 否 | compute 操作 |
| `VOLCENGINE_COMPUTE_MIN_CU` / `VOLCENGINE_COMPUTE_MAX_CU` | 否 | compute 弹性上下限 |
| `VOLCENGINE_TAG_KEY` / `VOLCENGINE_TAG_VALUE` | 否 | 标签管理 |
| `VOLCENGINE_ALLOW_IPS` | 否 | `network update` 白名单 |
| `VOLCENGINE_SCHEMA_DIFF_JOB_ID` | 否 | `schema-diff` 系列 |
| `VOLCENGINE_SQL` | 否 | `db query` / `psql -- -c` 语句 |
| `VOLCENGINE_DUMP_OUTPUT` | 否 | `db dump` 输出路径 |
| `VOLCENGINE_PULL_FILE` | 否 | `db pull` 输出路径 |

### 4.3 命令注释语义

- **只读**：可以直接放开执行，不会修改远端资源。
- **具有副作用**：会修改配置、账号、网络策略或写本地文件。
- **破坏性**：会释放资源（`delete` / `drop` / `stop` / `restore` 等），生产环境务必谨慎。

---

## 5. 各分组说明

### 5.1 Authentication（`script/authentication/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `configure.sh` | `configure`、`configure set/get/list` | `set` 默认注释；会修改 `~/.volcengine/config.json`，且会把 current profile 切到该 profile |
| `login.sh` | `login`（AK-SK 模式）、`login --remote`（Console Login）、`login --skip-region` | 全部默认注释；`--remote` 需要人工授权码 |
| `logout.sh` | `logout` | 默认注释；会清理本地登录缓存 |
| `status.sh` | `status`、`status -o json` | 全部只读，默认放开 |

### 5.2 Discovery（`script/discovery/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `branches.sh` | `list` / `get` / `default` / `children` / `create` / `update` / `delete` / `restart` / `set-default` / `restore-window` / `restorable` / `restore` / `diff`（共 13 个子命令） | 只读子命令默认放开；写类默认注释 |
| `operations.sh` | `operations list`（含 alias `operation` / `ops`） | 只读 |
| `projects.sh` | `projects list`（含 `-o json`） | 只读 |
| `workspaces.sh` | `list` / `get` / `create` / `delete` / `start` / `stop` / `rename` / `deletion-protection` / `compute-settings` / `settings` / `overview`（共 11 个子命令） + `workspace` alias | 只读子命令默认放开；写类默认注释 |

### 5.3 Database Access（`script/database-access/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `connection-string.sh` | 主命令、带 `--role-name` 变体、`--reveal-password` 明文变体、`cs` alias | 明文密码变体默认注释 |
| `db.sh` | `db query` / `db dump` / `db pull` / `db advisors` | `query` 默认给出只读 SELECT 示例；`dump` / `pull` / 危险 DDL 默认注释 |
| `inspect.sh` | `inspect db` 下 13 个子命令：`db-stats` / `index-stats` / `table-stats` / `role-stats` / `long-running-queries` / `locks` / `blocking` / `replication-slots` / `vacuum-stats` / `traffic-profile` / `outliers` / `calls` / `bloat` | 全部只读，依赖 `pg_stat_statements` 的子命令需要预先启用扩展 |
| `psql.sh` | 默认分支进入、指定分支进入、`-- -c "SQL"` 非交互执行 | 默认注释（会开交互式 session） |
| `schema-diff.sh` | `status` / `result` / `download` | 全部只读 |

### 5.4 Database Resources（`script/database-resources/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `computes.sh` | `list` / `get` / `create` / `update` / `delete` / `restart` / `enable-ap` | 只读默认放开；写类默认注释 |
| `databases.sh` | `list` / `create` / `delete` + `database` alias | 只读默认放开；写类默认注释 |
| `endpoints.sh` | `endpoints list` + `endpoint` alias、按 branch 过滤 | 只读 |
| `roles.sh` | `list` / `create` / `delete` / `reset-password` + `role` / `accounts` / `account` alias | 密码禁写死；写类默认注释 |

### 5.5 Workspace Settings（`script/workspace-settings/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `network.sh` | `network get` / `network update` | `update` 默认注释（可能影响外部访问） |
| `tags.sh` | `tags list` / `tags add` / `tags remove` | 写类默认注释 |

### 5.6 Tools（`script/tools/`）

| 脚本 | 覆盖命令 | 备注 |
|---|---|---|
| `completion.sh` | `completion bash` / `zsh` / `fish` / `powershell` | 只读；重定向到 shell rc 的变体默认注释 |
| `config.sh` | `config` / `config status` | 只读 |
| `help.sh` | `help`、`--help`、以及各一级命令的 `help X` | 只读 |
| `mcp.sh` | `mcp` / `mcp serve` | `serve` 会长期占用 stdio，默认注释 |
| `update.sh` | `update` | 会替换本机二进制，默认注释 |
| `version.sh` | `version` / `--version` | 只读 |
