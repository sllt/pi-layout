# Pi Layout

基于 [Pi](https://github.com/sllt/pi) 框架的企业级 Go 应用脚手架（项目骨架）。

项目由 `kite-layout` 更名而来。本隔离候选固定依赖 `github.com/sllt/pi v0.2.4`，
不使用 go.work 或 replace；验收通过本地 module proxy 获取候选归档并生成校验和。
该版本尚未推送或打 tag，公开下载需等待框架与脚手架的配套发布。
候选保留旧 Migration API；主开发工作区的 Migration v2 属于 v0.3.0。

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
- `internal/server`：HTTP / migration / task 的运行时注册
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
go run ./cmd/migration
```

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

- `cmd/server` 使用 Fx 负责依赖装配，启动后通过 `piApp.RunContext(ctx)` 交给 Pi 管理 HTTP/gRPC/metrics 生命周期。
- `cmd/server` 自己创建 signal context，避免 Fx `Run()` 和 Pi `Run()` 双重接管 OS signal。
- Fx 只调用 `Start` / `Stop`，Pi app 由 `RunContext` 在同一个 context 下启动、阻塞和优雅停机。
- `cmd/migration` 是一次性入口，仍通过 Fx `OnStart` 执行迁移并主动 shutdown。
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
