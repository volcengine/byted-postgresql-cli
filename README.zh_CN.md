# 火山引擎 PostgreSQL CLI

[English](README.md) | 中文

`byted-postgresql-cli` 是火山引擎 PostgreSQL 云平台的命令行客户端，用于管理
Workspace、Branch、Compute、Database、Role、Endpoint、Operation、Schema Diff
以及 PostgreSQL 数据面和 MCP 工作流。

## 环境要求

- 使用 npm 安装 CLI 需要 Node.js 和 npm
- 需要具备访问 AIDAP PostgreSQL API 的火山引擎账号
- `db dump` 和 `db pull` 需要本机安装 `pg_dump`
- `psql` 和相关 PostgreSQL 数据面命令需要本机安装 `psql`

## 安装

通过 npm 安装 CLI：

```bash
npx @byted-postgresql/cli@latest install 
byted-postgresql-cli --help
```

更新已安装的 CLI：

```bash
npm update --global @byted-postgresql/cli
```

## 开发

从源码构建并运行仓库测试：

```bash
make build
byted-postgresql-cli --help
make test
make vet
npm run test:npm
```

构建支持的发布平台：

```bash
make release
```

## 认证

支持浏览器 Console Login 和 AK/SK Profile：

```bash
byted-postgresql-cli login --region cn-beijing
byted-postgresql-cli configure set \
  --profile default \
  --access-key '<access-key-id>' \
  --secret-key '<secret-access-key>' \
  --region cn-beijing
```

自动化环境可以使用环境变量：

```bash
export VOLCENGINE_ACCESS_KEY='<access-key-id>'
export VOLCENGINE_SECRET_KEY='<secret-access-key>'
export VOLCENGINE_SESSION_TOKEN='<session-token>'
export VOLCENGINE_REGION='cn-beijing'
```

请勿提交凭证、在脚本中写入凭证，或将凭证打印到 CI 日志中。

## 常用命令

```bash
CLI=byted-postgresql-cli

$CLI --region cn-beijing workspaces list
$CLI --region cn-beijing branches list --workspace-id <workspace-id>
$CLI --region cn-beijing computes list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id>
$CLI --region cn-beijing databases list \
  --workspace-id <workspace-id> \
  --branch-id <branch-id>
```

列表命令默认返回 10 条记录，可以使用 `--offset` 和 `--limit` 获取后续分页。
结构化输出支持 `table`、`json`、`yaml`、`csv` 和 `tsv`。

## PostgreSQL 数据面

获取脱敏连接串：

```bash
$CLI --region cn-beijing connection-string \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

执行 SQL、进入 `psql`、导出数据库和拉取 Schema：

```bash
$CLI --region cn-beijing db query \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name> \
  --sql 'select now()'

$CLI --region cn-beijing db dump --dry-run \
  --workspace-id <workspace-id> \
  --branch-id <branch-id> \
  --database-name <database-name> \
  --role-name <role-name>
```

默认会掩码连接密码。只有在受信任环境中确有需要时，才使用 `--reveal`，
并避免将敏感输出保存到日志或 Shell 历史。

## MCP

通过 stdio 启动 MCP Server：

```bash
$CLI --region cn-beijing mcp serve \
  --workspace-id <workspace-id> \
  --read-only
```

为 AI 助手使用时，建议启用 `--read-only` 并限制到明确的 Workspace。

## 本地数据

- 火山引擎凭证：`~/.volcengine/config.json`
- Console Login 缓存：`~/.volcengine/login/cache/`
- CLI 安装和更新数据：`~/.volcengine-postgresql/`

请为上述目录设置正常的用户私有权限。

## 贡献和行为准则

当前项目由火山引擎 PostgreSQL 团队维护，暂不接受外部代码贡献。
请阅读 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

## 安全

安全问题请阅读 [SECURITY.md](SECURITY.md)，不要通过公开 Issue 披露漏洞。

## 许可证

本项目使用 [MIT License](LICENSE)。
