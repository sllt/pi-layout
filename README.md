# Pi Layout

基于 [Pi](https://github.com/sllt/pi) 框架的企业级 Go 应用脚手架（项目骨架）。

项目由 `kite-layout` 更名而来，使用已发布的 `github.com/sllt/pi v0.3.0`。普通项目无需本地框架源码：

```sh
GOWORK=off go mod download
GOWORK=off go build ./...
```

开发框架本身时可自行使用忽略提交的 go.work；发布验收使用 `GOWORK=off`。

## 特性

- **分层架构** - Handler → Service → Repository → Model
- **依赖注入** - 使用 [Fx](https://github.com/uber-go/fx) 管理依赖图与入口装配
- **数据库支持** - 基于 Pi 的 SQL 抽象，支持事务
- **JWT 认证** - 全局解析令牌 + 受保护路由强制鉴权
- **定时任务** - 集成 gocron 调度
- **数据库迁移** - 内置迁移入口
- **可观测性** - 集成 OpenTelemetry 与 Prometheus

## 目录概览

- `cmd/server`：HTTP 服务入口（gRPC 用户示例默认不注册）
- `cmd/migration`：数据库迁移入口
- `cmd/task`：定时任务入口
- `internal/handler`：接口层（协议适配）
- `internal/service`：业务逻辑层
- `internal/repository`：数据访问层
- `internal/router`：路由与鉴权包装
- `internal/server`：HTTP / task 的运行时注册；`internal/migrationcmd`：一次性迁移命令
- `internal/bootstrap`：Fx modules 与基础构造函数
- `docs/architecture`：架构与业务模块开发约定
- `pkg/config`：环境配置约定

## 快速开始

### 1) 准备配置

```bash
cp configs/.env.example configs/.env
```

然后修改 `configs/.env` 中的关键项（至少包括 `JWT_SECRET`）。

> 注意：`configs/.env` 已被 `.gitignore` 忽略，不应提交真实密钥。

### 2) 初始化依赖工具（可选）

```bash
make init
```

### 3) 执行数据库迁移

```bash
go run ./cmd/migration plan
go run ./cmd/migration up --timeout=10m --lock-ttl=15m
go run ./cmd/migration status
```

`up` 是默认子命令，默认开启迁移锁；显式 `--lock=false` 才关闭。
支持 `--target=<version>`、`up --dry-run`、`--config-dir=configs`。输出单条 JSON 摘要到 stdout，
日志到 stderr；错误、取消、超时、gap 均返回非零退出码。plan/status 不取得执行锁，可能创建状态表。
锁无自动续租，`--lock-ttl` 必须大于 `--timeout`，迁移函数必须配合 context 取消。

随项目提供的 users/user_profiles DDL 仅支持 SQLite，命令会拒绝其他方言；
切换 MySQL/PostgreSQL 时应先修改业务 DDL 与此方言检查。Pi 框架自身支持这些迁移后端。
完整说明见 [迁移命令](docs/migration.md)。

如需新增迁移模板：

```bash
pi migrate create add_users_index
```

### 4) 启动服务

```bash
go run ./cmd/server
# 或使用 nunu
nunu run ./cmd/server
```

### 5) 启动任务调度

```bash
go run ./cmd/task
```

## 常用命令

```bash
make test     # 运行测试并生成覆盖率报告
make build    # 构建 server 二进制
make swag     # 生成 swagger 文档
make bootstrap
```

`make bootstrap` 会启动 docker-compose、执行迁移并运行服务。

## 配置约定

- Pi 默认从项目根目录下的 `./configs/.env` 读取配置
- 本地开发先复制 `configs/.env.example` 到 `configs/.env`
- **密钥管理建议**：
  - 本地开发：放 `configs/.env`（不提交）
  - CI/CD：放平台 Secret
  - 生产环境：使用环境变量或密钥管理系统注入

## 生命周期约定

### v0.3.1 升级

JWT 只从 `Authorization: Bearer <token>` 接受凭证，cookie/query/raw token 不再被识别。
新 token 要求 HS256、有效 exp、配置的 issuer/audience 和一致的 subject/userId。
`JWT_SECRET` 至少 32 字节且不能是示例值；`JWT_TTL` 默认 1h，`JWT_ISSUER` 默认 pi-layout，`JWT_AUDIENCE` 默认 pi-api。
旧版缺少这些 claims 的 token 从本版起失效，升级服务后客户端需重新登录；不提供无截止日期的宽松解析兼容开关。
`jwt.NewJwt` 现在返回 `(*JWT, error)`，Fx 会展示配置错误；自定义构造代码需处理 error，业务发 token 使用 `Issue(userID)`。

CORS 已集中到 Pi 框架，layout 不再叠加反射 Origin 的 middleware；需要跨域时设置 `CORS_ALLOWED_ORIGINS`。
端口可用 `HTTP_ADDR=127.0.0.1:0` 申请随机端口，`*_ENABLED=false` 明确禁用；TLS 配置错误不会回落明文。

- `cmd/server` 使用 Fx 负责依赖装配，启动后通过 `piApp.RunContext(ctx)` 交给 Pi 管理 HTTP/gRPC/metrics 生命周期。
- `cmd/server` 自己创建 signal context，避免 Fx `Run()` 和 Pi `Run()` 双重接管 OS signal。
- Fx 只调用 `Start` / `Stop`，Pi app 由 `RunContext` 在同一个 context 下启动、阻塞和优雅停机。
- `cmd/migration` 是一次性入口，通过 `run(ctx) error` 传递失败，仅打开所需 SQL 连接；每次执行均关闭连接，不启动 HTTP/gRPC/metrics。
- `cmd/task` 由 Fx 托管 gocron scheduler 生命周期。

## 鉴权策略

- 全局中间件使用 `NoStrictAuth`：有 token 就解析 claims，无 token 不拦截。
- 需要登录的路由在 `internal/router` 使用 `RouteGroup.UseMiddleware(...)` 做分组强制鉴权。

### gRPC 用户示例的状态

`internal/grpc/user` 保留为协议适配代码参考，默认不注册 `UserService`。只配置
`GRPC_PORT` 不会启用该服务；当前默认入口也不会启动 gRPC 监听。
此前依赖这些 RPC 的应用需要迁移到已鉴权的 HTTP 接口，或先完成以下条件：

- 从经过验证的凭证建立调用者身份；本人资料接口使用该身份确定用户。
- 访问其他用户资料时执行明确的资源授权，不能仅信任请求中的 `UserId`。
- 对齐 HTTP 与 gRPC 的业务输入校验、错误映射，并在业务层执行授权。
- 通过无令牌、无效令牌、用户 A 访问 A、A 访问 B、管理员访问 B 的权限测试。

这项关闭是临时缓解；跨协议身份与授权计划在 Pi v0.4.1 完整验收。

## 业务模块开发

新增业务模块建议参考 `docs/architecture/module.md`。当前 `user` 模块已经演示：

- API DTO、domain types、model、repository、service、handler、router 的分层边界；
- 注册用户时在一个事务内同时创建账号记录和用户资料记录；
- 更新资料时在一个事务内同时更新账号邮箱和资料昵称。

错误码与响应 envelope 约定参考 `docs/architecture/error.md`。Handler / Service 直接返回 `pkg/errcode`，`net/http` middleware 使用 `errcode.WriteHTTPError`，避免重复手写 JSON 响应。

## 测试

```bash
make test
```

接口冒烟测试（会自动执行迁移、启动服务并验证核心用户接口）：

```bash
bash scripts/smoke.sh
```

当前包含 service/repository 测试；handler 测试依赖 Pi 的集成测试上下文。
