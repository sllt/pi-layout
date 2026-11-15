# Pi Layout

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev/)
[![Pi Framework](https://img.shields.io/badge/Pi-v0.5.0-blue)](https://github.com/sllt/pi)

[English](./README.md) | 简体中文

**Pi Layout** 是 [Pi](https://github.com/sllt/pi) 框架的官方生产级 Go 应用脚手架（项目骨架）。它将 Uber [Fx](https://github.com/uber-go/fx) 依赖注入与 Pi 的 `pi.Build` 显式生命周期、泛型类型化 HTTP 契约、OpenAPI 3.0.3 自动导出、gRPC 适配层、带锁数据库迁移及受监督 Cron 定时任务有机结合。

---

## 核心特性

- **清晰的分层架构** — 严格划分 `API DTO / Proto → Handler → Service → Repository → Model` 职责边界，在 HTTP、gRPC 与后台任务之间复用统一的业务校验与授权逻辑。
- **显式 Fx + `pi.Build` 生命周期** — Fx 构造期仅分配无副作用的稳定句柄；`OnStart` 按序激活受管 SQL 与 Pi 运行时，`OnStop` 等待在途请求与任务排空后再逆序释放资源。
- **类型化 HTTP 契约与 OpenAPI 3.0.3** — 路由层采用 `pi.Register` 泛型注册与 `pi.Meta` DTO 标签，构造期通过 `app.RouteError()` 预检路由与校验规则。`cmd/openapi` 无需 `swag` 注释即可直接生成确定性的 `docs/openapi.json` 与 `docs/openapi.yaml`；设置 `OPENAPI_ENABLED=true` 可在线开启规范端点与内置离线 Scalar API 文档界面（`/.well-known/swagger`）。
- **统一身份与业务层授权** — 传输层仅从已验证的 `Authorization: Bearer <token>`（强制校验 `HS256`、`iss`、`aud`、`exp` 与 `sub`）建立 `pi/auth.Principal` 主体；Service 层统一执行本人或管理权限（`users:read:any`、`users:write:any`）判定。
- **安全 SQL 事务与一次性迁移入口** — Repository 基于 `BeginTxContext` 封装事务及嵌套回滚保护；`cmd/migration` 提供带锁迁移执行与状态检查（`up`、`plan`、`status`、`--dry-run`），不启动任何网络端口。
- **受监督 Cron 定时任务** — `cmd/task` 基于 Pi 内置 Actor 受监督 Cron 调度器运行（默认 `CronSkipIfRunning` 防重入、`time.UTC` 时区），支持优雅停机排空。
- **可重复代码生成与容器化交付** — `make generator` 先在临时隔离目录生成、格式化并编译验证 Protobuf、gRPC wrapper、Mock 与 OpenAPI 产物后再落盘（自动保留手写 `userservice_server.go`）；根目录 `Dockerfile` 构建非 root（`UID 10001`）多入口最小运行镜像（`server` / `task` / `migration`）。

---

## 快速开始

### 方式一：通过 `pi` CLI 初始化新项目（推荐）

```bash
go install github.com/sllt/pi/cmd/pi@v0.5.0
pi init --module example.com/acme/orders ./orders
cd ./orders
```

### 方式二：直接运行本仓库

#### 1. 准备配置文件

```bash
mkdir -p storage
cp configs/.env.example configs/.env
```

编辑 `configs/.env`，将 `JWT_SECRET` 修改为至少 32 字节的随机密钥（启动时会拒绝默认示例值）。

#### 2. 执行数据库迁移

```bash
go run ./cmd/migration plan
go run ./cmd/migration up --timeout=10m --lock-ttl=15m
go run ./cmd/migration status
```

> 示例内置的 `users` / `user_profiles` 表结构默认针对 SQLite（`storage/app.db`）；如需切换至 PostgreSQL 或 MySQL，请同步调整业务迁移 DDL 与方言检查。

#### 3. 启动 HTTP 服务

```bash
go run ./cmd/server
```

如需开启在线 OpenAPI 3.0.3 规范端点（`/.well-known/openapi.json`、`/.well-known/openapi.yaml`）与内置离线 Scalar 文档页（`/.well-known/swagger`），可在环境变量或 `configs/.env` 中设置 `OPENAPI_ENABLED=true`。

#### 4. 启动定时任务进程（可选）

```bash
go run ./cmd/task
```

---

## 目录结构

```text
pi-layout/
├── api/
│   ├── proto/user/          # Protobuf 定义及生成的 gRPC 代码
│   └── v1/                  # 类型化 HTTP 请求/响应 DTO（基于 pi.Meta）
├── cmd/
│   ├── migration/           # 一次性数据库迁移命令行入口
│   ├── openapi/             # OpenAPI 3.0.3 离线导出与差异检查入口
│   ├── server/              # HTTP（及可选 gRPC）服务入口
│   └── task/                # 受监督 Cron 定时任务进程入口
├── configs/                 # .env.example 配置模板（本地 configs/.env 已忽略提交）
├── deploy/docker-compose/   # 可选本地 MySQL / Redis Compose Profiles
├── docs/                    # 生成的 openapi.json 与 openapi.yaml 契约文件
├── internal/
│   ├── bootstrap/           # Fx 模块装配、Pi Build 构造与 OpenAPI 配置
│   ├── grpc/user/           # 生成的 Pi gRPC wrapper 与手写业务适配层
│   ├── handler/             # HTTP 协议转换处理器
│   ├── middleware/          # JWT Principal 提取中间件
│   ├── migrationcmd/        # 带锁迁移命令执行器
│   ├── model/               # 数据库持久化模型
│   ├── repository/          # SQL 数据访问与事务封装
│   ├── router/              # 路由分组、强制鉴权中间件与 OpenAPI 文档属性
│   ├── server/              # HTTP、gRPC 与 Cron 任务注册
│   ├── service/             # 核心领域逻辑、参数校验与 Principal 授权
│   ├── task/                # 定时后台任务实现
│   └── types/               # 协议无关的领域层数据结构
├── migrations/              # 版本化数据库迁移定义
├── pkg/                     # 公共基础包（config / errcode / jwt / log / sid）
├── scripts/                 # 可重复生成、端到端冒烟测试与容器验收脚本
└── test/                    # 集成测试与生成的 Mock
```

---

## 配置与安全约定

- **JWT 鉴权** — 仅接受 `Authorization: Bearer <token>` 请求头，不从 query 或 cookie 读取凭证。支持配置 `JWT_SECRET`（≥ 32 字节）、`JWT_TTL`（默认 `1h`）、`JWT_ISSUER`（默认 `pi-layout`）与 `JWT_AUDIENCE`（默认 `pi-api`）。
- **CORS 与 TLS** — 默认关闭跨域访问，按需通过 `CORS_ALLOWED_ORIGINS` 显式配置完整来源列表。HTTP（`CERT_FILE` / `KEY_FILE`）与 gRPC（`GRPC_CERT_FILE` / `GRPC_KEY_FILE`）证书配置错误会在启动期立即报错拒绝，绝不静默降级为明文。
- **gRPC 示例开关** — 示例 `UserService` 默认关闭，仅在同时显式设置 `GRPC_ENABLED=true` 与 `USER_GRPC_ENABLED=true` 时注册启用。
- **统一错误包络** — `pkg/errcode.Error` 直接复用 Pi 的 `apperror.Error`，在 HTTP JSON 响应、gRPC 状态码与命令行退出码之间保持一致映射。

---

## 常用开发命令

```bash
make init            # 安装固定版本的 pi、mockgen、air 与 protoc 插件到 .tools/bin
make build           # 编译 server、task、migration、openapi 四个二进制到 ./bin
make test            # 运行全部单元与集成测试
make race            # 开启 -race 竞态检测运行全部测试
make openapi         # 重新生成 docs/openapi.json 与 docs/openapi.yaml
make generator       # 在隔离目录安全重生成 Protobuf、gRPC wrapper、Mock 与 OpenAPI 规范
make check-generated # 检查生成物是否与源码保持一致（零漂移）
make smoke           # 运行真实二进制冒烟验收（迁移、鉴权、状态码断言与 SIGTERM 优雅退出）
make container       # 构建并验收多入口容器镜像
```
