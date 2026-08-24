# Volcengine PostgreSQL CLI 产品文档

> **Volcengine PostgreSQL CLI（`byted-postgresql-cli`）是面向火山引擎 PostgreSQL 云平台的命令行工具。它将 PostgreSQL Workspace、Branch、Compute、Database、Role、Endpoint 等云资源管理能力，与 SQL 查询、`psql`、`pg_dump`、Schema Diff 和 MCP 能力统一到一个命令行入口，服务于开发、运维、数据管理、CI/CD 和 AI Agent 自动化场景。**

## 产品概述

Volcengine PostgreSQL CLI 是火山引擎 PostgreSQL 产品的专用 CLI，不是官方 PostgreSQL 客户端的简单包装，也不负责管理其他云厂商资源。CLI 通过火山引擎 AIDAP 管控面管理云资源，并在需要时解析 PostgreSQL 数据库连接信息，调用 PostgreSQL 原生工具或驱动访问数据面。

| 维度 | 说明 |
|-|-|
| 产品名称 | Volcengine PostgreSQL CLI |
| 命令名称 | `byted-postgresql-cli` |
| 目标用户 | PostgreSQL 开发者、DBA、DevOps、QA、平台工程师和 AI Agent |
| 核心价值 | 用统一命令行完成云资源管理、数据库访问、Schema 操作和自动化集成 |
| 管理对象 | Workspace、Branch、Compute、Database、Role、Endpoint、Operation |
| 数据面能力 | SQL 查询、`psql` 会话、数据库导出、Schema 拉取、数据库诊断 |
| 自动化能力 | 表格及结构化输出、分页、非交互式参数、MCP over stdio |
| 技术栈 | Go、Cobra、Volcengine Go SDK、pgx、PostgreSQL 客户端工具、MCP SDK |
| 分发方式 | npm 包 `@byted-postgresql/cli`，按平台分发原生二进制 |
| 许可证 | MIT License |

## 产品价值

- **资源统一管理**：围绕 PostgreSQL Workspace 组织 Branch、Compute、Database、Role 和 Endpoint。
- **控制面与数据面贯通**：既可以查询和修改云资源，也可以直接执行 SQL、进入 `psql` 或导出数据库。
- **自动化友好**：支持 `table`、`json`、`yaml`、`csv` 和 `tsv` 输出，列表接口提供 `limit`、`offset` 分页参数。
- **安全默认**：连接串和 `pg_dump` 预览默认掩码密码；删除、停止等高风险操作默认要求确认。
- **AI Agent 可集成**：CLI 可以作为 Skill 或子进程工具使用，也可以通过 MCP Server 以 stdio 方式接入 AI 助手。

## 适用场景

- 创建、查询、启停和删除 PostgreSQL Workspace。
- 创建和管理 Branch，查询默认 Branch、子 Branch 及恢复相关信息。
- 管理 Compute 的规格、角色、重启和分析加速配置。
- 创建数据库、管理 PostgreSQL Role 和密码。
- 查询或配置 Workspace/Branch Endpoint 及网络设置。
- 在本地、CI/CD 或脚本中执行 SQL、导出数据库和拉取 Schema。
- 通过 Schema Diff 检查两个 Branch 的 Schema 差异。
- 通过 MCP 将 PostgreSQL 资源查询能力提供给 AI Agent。

## 当前边界

- CLI 面向火山引擎 PostgreSQL，不提供 Supabase、ByteCloud 或其他云厂商的 Provider 切换。
- `db dump` 和 `db pull` 依赖本机安装 `pg_dump`；`psql` 命令依赖本机安装 `psql`。
- CLI 不替代完整的 PostgreSQL 管理工具。需要复杂交互式 SQL、客户端扩展或高级备份恢复能力时，应直接使用 PostgreSQL 原生工具。
- MCP 当前定位为 PostgreSQL 资源发现和查询入口；是否允许写操作由 MCP Server 的工具集合和 `--read-only` 配置共同约束。
- 非交互式环境不能依赖 Workspace 或 Branch 的交互选择，应显式提供 `--workspace-id`、`--branch-id` 和 `--region`。

---

# 认证与配置

## 认证方式

CLI 支持浏览器 Console Login 和 AK/SK Profile 两类认证方式。

| 方式 | 适用场景 | 说明 |
|-|-|-|
| Console Login | 本地开发、交互式运维 | 浏览器完成 OAuth 2.0 + PKCE 登录，CLI 使用临时 STS 凭证 |
| AK/SK Profile | 本地脚本、CI/CD、Agent | 使用 Access Key、Secret Access Key，可选 Session Token |
| 环境变量 | 临时认证、容器和 CI/CD | 使用 `VOLCENGINE_*` 或兼容的 `VOLC_*` 环境变量 |

## Console Login

```bash
byted-postgresql-cli login --region cn-beijing
byted-postgresql-cli status
byted-postgresql-cli logout
```

`login` 会打开浏览器完成 Console Login。登录成功后，CLI 保存 Console Login profile，并在需要访问 API 时自动使用或刷新临时 STS 凭证。`logout` 只清理 Console Login profile 和缓存，不删除独立配置的 AK/SK profile。

## AK/SK Profile

```bash
byted-postgresql-cli configure set \
  --profile default \
  --access-key '<access-key-id>' \
  --secret-key '<secret-access-key>' \
  --region cn-beijing

byted-postgresql-cli configure get --profile default
byted-postgresql-cli configure list
```

`configure` 支持：

| 命令 | 说明 |
|-|-|
| `configure set` | 创建或覆盖一个 AK/SK profile |
| `configure get` | 查看 profile，敏感字段脱敏 |
| `configure list` | 列出本地 profile |
| `configure delete` | 删除 profile |

## 环境变量与优先级

自动化环境可以使用：

```bash
export VOLCENGINE_ACCESS_KEY='<access-key-id>'
export VOLCENGINE_SECRET_KEY='<secret-access-key>'
export VOLCENGINE_SESSION_TOKEN='<session-token>'
export VOLCENGINE_REGION='cn-beijing'
```

凭证优先使用环境变量；未提供环境变量时使用选中的本地 profile。Profile 选择支持全局 `--profile`、`VOLC_PROFILE` 和配置文件中的 current profile。Region 支持全局 `--region`、环境变量、profile 配置和默认 Region，默认 Region 为 `cn-beijing`。

## 本地数据

| 数据 | 默认路径 |
|-|-|
| Volcengine credential profiles | `~/.volcengine/config.json` |
| Console Login 临时凭证缓存 | `~/.volcengine/login/cache/` |
| CLI 安装、更新和产品级临时数据 | `~/.volcengine-postgresql/` |

CLI 与 Volcengine credential profile 使用产品级目录隔离：共享 Provider 认证配置，但安装状态、更新状态和 PostgreSQL CLI 临时数据不混入其他产品目录。

---

# 命令体系

## 全局参数

```text
--region <region>       指定火山引擎 Region
--profile <name>        指定 credential profile
-o, --output <format>   table、json、yaml、csv 或 tsv
--debug                 开启调试日志
```

自动化调用建议始终显式指定 `--region`、`--workspace-id` 和 `--branch-id`，并使用 `--output json`。

## Workspace

Workspace 是 PostgreSQL 云资源的顶层管理对象。

| 命令 | 说明 |
|-|-|
| `workspaces list` | 分页列出 Workspace，支持名称和 Project 过滤 |
| `workspaces get <workspace-id>` | 查询 Workspace 详情 |
| `workspaces create` | 创建 PostgreSQL Workspace |
| `workspaces delete <workspace-id>` | 删除 Workspace，级联影响其 Branch、Compute、Database 和 Endpoint |
| `workspaces start [workspace-id]` | 启动已停止的 Workspace |
| `workspaces stop [workspace-id]` | 停止运行中的 Workspace |
| `workspaces rename [workspace-id]` | 修改 Workspace 名称 |
| `workspaces deletion-protection [workspace-id]` | 开启或关闭删除保护 |
| `workspaces compute-settings [workspace-id]` | 修改 Workspace 自动伸缩和自动暂停设置 |
| `workspaces settings [workspace-id]` | 修改 Workspace 级设置 |
| `workspaces overview` | 查询 Workspace 状态聚合信息 |

创建 Workspace 时可以指定 Project、Compute 范围、自动暂停、保留时长、网络协议、VPC、Subnet、删除保护和 tags。

```bash
byted-postgresql-cli --region cn-beijing workspaces create \
  --name <workspace-name> \
  --project-name <project-name>
```

## Project

```bash
byted-postgresql-cli --region cn-beijing projects list --output json
```

`projects list` 用于查看当前账号可见的计费 Project 及其 Workspace 数量。Project 是资源归属和计费维度，不是 PostgreSQL 数据面连接目标。

## Branch

Branch 是 Workspace 下的 PostgreSQL 分支资源，可用于开发、测试、预发布和隔离验证。

| 命令 | 说明 |
|-|-|
| `branches list` | 列出 Workspace 下的 Branch |
| `branches get <branch-id>` | 查询 Branch 详情 |
| `branches create` | 创建 Branch，可指定父 Branch 和时间点 |
| `branches update <branch-id>` | 修改 Branch 名称或保护状态 |
| `branches delete <branch-id>` | 删除 Branch |
| `branches default` | 查看默认 Branch |
| `branches set-default <branch-id>` | 设置默认 Branch |
| `branches children` | 查询子 Branch |
| `branches restart <branch-id>` | 重启 Branch 的 Compute |
| `branches restore-window <branch-id>` | 查询时间点恢复窗口 |
| `branches restorable` | 查询指定时间点可恢复的 Branch |
| `branches restore` | 执行时间点恢复 |
| `branches diff` | 发起两个 Branch 的 Schema 比较 |

Branch 相关命令通常需要 `--workspace-id`。未显式提供 `--branch-id` 时，部分命令会使用 Workspace 默认 Branch；在非 TTY 环境中不应依赖交互选择。

## Compute

| 命令 | 说明 |
|-|-|
| `computes list` | 列出 Workspace/Branch 下的 Compute |
| `computes get <compute-id>` | 查询 Compute 详情 |
| `computes create` | 创建 Compute |
| `computes update <compute-id>` | 修改名称、最小 CU 和最大 CU |
| `computes delete <compute-id>` | 删除 Compute |
| `computes restart <compute-id>` | 重启 Compute 所属 Branch |
| `computes enable-ap <compute-id>` | 开启分析加速配置 |

## Database 与 Role

```bash
byted-postgresql-cli --region cn-beijing databases list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --output json

byted-postgresql-cli --region cn-beijing roles list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --output json
```

| 命令组 | 能力 |
|-|-|
| `databases list` | 列出 Branch 下的 Database |
| `databases create` | 创建 Database，可指定描述和拥有者 Role |
| `databases delete <name>` | 删除 Database |
| `roles list` | 列出 Branch 下的 PostgreSQL Role/Account |
| `roles create` | 创建 Role 并设置密码 |
| `roles delete <role-name>` | 删除 Role |
| `roles reset-password <role-name>` | 重置 Role 密码 |

数据库数据面命令可以通过 `--database-name` 和 `--role-name` 指定目标。对于只有一个可用 Database 或 Role 的场景，交互式命令可以自动解析；脚本中建议始终显式传参。

## Endpoint 与网络

| 命令 | 说明 |
|-|-|
| `endpoints list` | 查询 Workspace/Branch Endpoint |
| `endpoints enable-public` | 开启公网 Endpoint |
| `endpoints disable-public` | 关闭公网 Endpoint |
| `endpoints enable-private` | 配置私网 Endpoint |
| `endpoints eips list` | 查询 EIP |
| `endpoints vpcs list` | 查询 VPC |
| `endpoints subnets list` | 查询 Subnet |
| `network ...` | 管理 Workspace 网络设置 |
| `tags ...` | 管理 Workspace tags |

公网、私网、EIP、VPC 和 Subnet 属于连接可达性配置，修改前应确认业务流量、访问控制和安全组策略。

## Operations 与 Status

```bash
byted-postgresql-cli --region cn-beijing operations list \
  --workspace-id <workspace-id> \
  --status <status> \
  --output json

byted-postgresql-cli --region cn-beijing status --output json
```

`operations list` 用于查询异步资源操作，可按 Workspace、Branch、状态和 Action 过滤。`status` 用于查看当前解析后的 Region、profile 和凭证状态，便于排查“命令访问了错误 Region 或 profile”的问题。

---

# PostgreSQL 数据面

## 连接目标

CLI 支持两类数据库目标：

1. **云资源目标**：通过 `--workspace-id`、`--branch-id`、`--database-name` 和 `--role-name` 定位 PostgreSQL 实例，并由控制面临时解析连接信息。
2. **显式连接串**：通过 `--db-url` 直接连接用户提供的 PostgreSQL 数据库。

解析到的密码、STS token 和连接串只在当前命令需要时使用，默认不输出或落盘。

## Connection String

```bash
byted-postgresql-cli --region cn-beijing connection-string \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

该命令用于输出 PostgreSQL 连接串。默认对密码进行掩码；只有在明确需要并确认输出环境可信时，才使用对应的明文输出能力。

## SQL Query

```bash
byted-postgresql-cli --region cn-beijing db query \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name> \
  --sql 'select now()' \
  --output json
```

`db query` 支持位置参数 SQL、`--file` 读取 SQL 文件，并支持结构化结果输出。Agent 或脚本调用时，应避免将不可信数据库内容直接拼接进后续 Shell 命令或提示词。

## psql

```bash
byted-postgresql-cli --region cn-beijing psql \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

`psql` 命令解析云端连接信息后启动本机 PostgreSQL `psql` 客户端，适用于需要交互式 SQL、事务控制或原生 `psql` 元命令的场景。

## Dump 与 Pull

预览 `pg_dump` 命令而不执行：

```bash
byted-postgresql-cli --region cn-beijing db dump --dry-run \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

执行导出：

```bash
byted-postgresql-cli --region cn-beijing db dump \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name> \
  --file ./backup.sql
```

`db dump` 支持 data-only、schema-only、roles-only 和 dry-run。dry-run 默认掩码密码；不要把 `--reveal` 输出写入日志、Shell 历史或构建产物。

`db pull` 使用 `pg_dump` 拉取 PostgreSQL Schema 到本地 SQL 文件。该命令用于生成 Schema 快照，不代表 CLI 会替用户执行远端 Migration。

## Database Advisors

```bash
byted-postgresql-cli --region cn-beijing db advisors \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name> \
  --output json
```

`db advisors` 执行独立的 PostgreSQL 健康检查，输出可操作的安全和性能问题。

## Inspect

`inspect db` 提供远程只读诊断能力，包括：

| 诊断类别 | 示例 |
|-|-|
| 查询与会话 | `calls`、`long-running-queries`、`role-connections` |
| 锁与阻塞 | `locks`、`blocking` |
| 表与索引 | `table-sizes`、`table-index-sizes`、`table-record-counts`、`index-stats` |
| 性能统计 | `db-stats`、`role-stats`、`outliers`、`seq-scans` |
| 维护状态 | `vacuum-stats` |
| 复制能力 | `replication-slots` |
| 空间与膨胀 | `bloat` |

`inspect report` 可以生成综合的远程只读数据库诊断报告。

---

# Schema Diff

Schema Diff 用于比较两个 Branch 上指定 Database 和 Schema 的结构差异，并通过异步 Job 生成结果。

```bash
byted-postgresql-cli --region cn-beijing branches diff \
  --workspace-id <workspace-id> \
  --source-branch-id <source-branch-id> \
  --source-database-name <source-database-name> \
  --source-schema-name public \
  --target-branch-id <target-branch-id> \
  --target-database-name <target-database-name> \
  --target-schema-name public
```

通过 Job ID 查看结果：

```bash
byted-postgresql-cli --region cn-beijing schema-diff status <job-id> \
  --workspace-id <workspace-id>

byted-postgresql-cli --region cn-beijing schema-diff result <job-id> \
  --workspace-id <workspace-id>

byted-postgresql-cli --region cn-beijing schema-diff download <job-id> \
  --workspace-id <workspace-id>
```

Schema Diff 适用于开发分支与目标分支之间的 Schema 评审、发布前检查和迁移 SQL 预览。它不等同于自动执行 Migration，结果应用前仍需经过人工或流水线审核。

---

# AI Agent 与 MCP 集成

## 集成方式

| 方式 | 状态 | 说明 |
|-|-|-|
| Skill | 已提供 | 仓库内 `skills/byted-postgresql/` 提供认证、资源查询、数据面和 MCP 使用约束 |
| Subprocess | 支持 | Agent 可直接启动 CLI，使用稳定参数和结构化输出 |
| MCP Server | 支持 | `mcp serve` 通过 stdio 提供 PostgreSQL 工具 |
| 内置 HTTP MCP | 不提供 | CLI 当前不直接启动 HTTP MCP 服务 |

## 启动 MCP

```bash
byted-postgresql-cli --region cn-beijing mcp serve \
  --workspace-id <workspace-id> \
  --read-only
```

`--workspace-id` 可以把工具范围限制到指定 Workspace；`--read-only` 用于以只读模式运行 MCP。MCP 使用与 CLI 相同的 Volcengine profile、Region 和凭证解析逻辑。

## MCP 工具范围

当前 MCP 资源工具覆盖：

- `workspaces_list`、`workspace_get`
- `branches_list`、`branch_get`
- `computes_list`
- `databases_list`
- `roles_list`
- `operations_list`

列表工具使用分页输入，返回包含资源列表、总数、当前 offset、limit 和下一页信息的结构化结果。建议 Agent 在工具调用中显式传入 Workspace ID，并根据 `total`、`limit`、`offset` 和 `next_offset` 继续分页。

## Agent 使用约束

- 自动化调用显式指定 `--region`、`--workspace-id` 和 `--branch-id`。
- 查询优先使用 `--output json`，不要解析人类可读表格。
- 大列表使用 `--limit` 和 `--offset`，避免一次性消耗过多上下文。
- 删除、停止、恢复、密码修改等写操作必须由调用方明确授权，并按命令要求传入确认参数。
- 不将密码、STS token、连接串和未经处理的数据库返回内容写入日志或生成提示词。

---

# 技术架构

## 请求路由

| 能力类型 | 认证 | 底层实现 |
|-|-|-|
| PostgreSQL 管控面 | AK/SK 或 Console Login STS | Volcengine Go SDK / AIDAP API |
| PostgreSQL 数据面 | 控制面临时解析的数据库凭证 | pgx、`psql` 或 `pg_dump` |
| Schema Diff | AK/SK 或 Console Login STS | AIDAP 异步 Schema Diff API |
| MCP | 启动进程时解析的同一套凭证 | MCP SDK + stdio transport |
| 显式 `--db-url` | 用户提供的连接信息 | PostgreSQL 直连 |

## 核心组件

| 组件 | 职责 |
|-|-|
| `cmd/root.go` | Cobra 根命令、全局参数、命令分组、配置解析入口 |
| `cmd/cmd_login.go` | Console Login、Logout |
| `cmd/cmd_configure.go` | AK/SK Profile 管理 |
| `internal/volcengine/config.go` | 凭证、Profile、Region 和 Endpoint 解析 |
| `internal/volcengine/console_login.go` | OAuth/PKCE 登录和 STS 刷新 |
| `internal/volcengine/client.go` | AIDAP SDK Client 和请求传输 |
| `internal/volcengine/workspaces.go` | Workspace 管控面操作 |
| `internal/volcengine/branches.go` | Branch 管控面操作 |
| `cmd/cmd_database_tools.go` | SQL、dump、pull、advisors 和数据库目标解析 |
| `internal/mcp/` | MCP Server、stdio transport 和 PostgreSQL 资源工具 |
| `internal/writer/` | table、JSON、YAML、CSV、TSV 输出 |
| `internal/pagination/` | 列表分页策略和分页元数据 |

## 关键设计

1. **专用产品命令树**：命令围绕 PostgreSQL 云资源设计，不保留不适用的其他产品命令。
2. **控制面与数据面分离**：资源管理使用 AIDAP；SQL、`psql` 和 `pg_dump` 使用 PostgreSQL 数据面。
3. **SDK 优先**：优先使用官方 Volcengine Go SDK 的强类型 API；必要时通过 AIDAP Action 访问尚未完整封装的能力。
4. **错误透传**：尽可能保留网关原始错误，帮助用户定位权限、资源状态和参数问题。
5. **分页和非 TTY 收敛**：列表默认限制返回量；非交互式运行时要求显式资源 ID，避免隐式选择。
6. **写操作安全**：删除、停止、恢复和其他非幂等操作默认进行确认或要求 `--yes`。
7. **敏感信息最小化**：凭证、数据库密码、连接串和临时 token 不默认展示或持久化。

---

# 安装与分发

## npm 安装

```bash
npx @byted-postgresql/cli@latest install
byted-postgresql-cli --help
```

更新已安装版本：

```bash
npm update --global @byted-postgresql/cli
```

npm 包通过轻量 JavaScript 启动器选择对应平台的原生 Go 二进制。当前发布矩阵覆盖：

| 操作系统 | 架构 |
|-|-|
| macOS | x86_64、arm64 |
| Linux | x86_64、arm64 |
| Windows | x86_64、arm64 |

## 源码构建

```bash
make build
make test
make vet
make test-npm
```

发布构建使用 `CGO_ENABLED=0` 和 `-trimpath`，并通过 `make release` 生成多平台二进制。

---

# 与其他 CLI 的关系

| 维度 | Volcengine PostgreSQL CLI |
|-|-|
| Provider | 固定为 Volcengine |
| 顶层资源 | PostgreSQL Workspace |
| 认证 | Console Login、AK/SK、STS 环境变量 |
| 管控面 | Volcengine AIDAP PostgreSQL API |
| 数据面 | PostgreSQL 原生连接、pgx、`psql`、`pg_dump` |
| 配置目录 | `~/.volcengine/` 与 `~/.volcengine-postgresql/` |
| CLI 包 | `@byted-postgresql/cli` |

本产品不应与其他 Provider 的 PostgreSQL CLI 共享命令语义、凭证目录或产品级状态目录。用户在自动化环境中应明确选择目标 CLI、Region、Profile 和资源 ID。

---

# 安全与合规

- 不在代码、示例、测试、日志、文档或构建产物中提交真实 AK/SK、Session Token、数据库密码、连接串、Cookie 或 Workspace/Branch 敏感标识。
- 示例统一使用 `<workspace-id>`、`<branch-id>`、`<access-key-id>` 等占位符。
- `configure get` 和连接串、dump 预览默认脱敏。
- 仅在受信任的本地环境中使用明文密码输出能力，且不得写入 Shell 历史或 CI 日志。
- 生产操作前核对 Region、Profile、Workspace ID、Branch ID 和命令的级联影响。
- 删除、停止、恢复、删除保护和密码修改等高风险操作应由用户或自动化审批流程明确授权。
- MCP 接入 AI Agent 时优先启用 `--read-only`，并将 Workspace 范围限制在业务必需范围。
- CLI 本身不保证数据库返回内容可信；Agent 应将数据库结果视为不可信外部数据，避免直接执行其中的命令或指令。

## 常见排查路径

1. 执行 `byted-postgresql-cli status --output json`，确认 Region、Profile 和凭证来源。
2. 执行 `workspaces list --output json`，确认账号对目标 Region 有访问权限。
3. 对资源命令显式补充 `--workspace-id`、`--branch-id`，排除交互选择和默认 Branch 的影响。
4. 对数据面命令确认本机存在 `psql` 或 `pg_dump`，并检查 Database、Role 和密码是否正确。
5. 使用 `--debug` 获取请求诊断信息，但不要把包含敏感信息的调试日志上传到公共位置。
