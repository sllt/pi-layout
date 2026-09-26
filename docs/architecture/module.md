# 业务模块开发约定

本文以当前 `user` 模块为蓝本，说明在 pi-layout 中新增业务模块时推荐遵循的分层方式。

## 分层职责

| 层级 | 目录 | 职责 |
| --- | --- | --- |
| API DTO | `api/v1` | 面向 HTTP 的请求/响应结构，保留 `json`、`binding`、Swagger 相关标签。 |
| Domain Types | `internal/types` | Service 层输入/输出结构，避免 Handler 直接把 HTTP DTO 传入业务层。 |
| Model | `internal/model` | 数据库存储结构，字段用 `db` tag 对齐表字段。 |
| Repository | `internal/repository` | 数据访问接口与实现，只处理 SQL、事务上下文和存储错误。 |
| Service | `internal/service` | 业务编排、事务边界、错误码映射、跨 Repository 协作。 |
| Handler | `internal/handler` | `pi.Context` 绑定参数，调用 Service，并把 output 转回 API DTO。 |
| Router | `internal/router` | 注册路由、路由分组和鉴权中间件。 |
| Migration | `migrations` | 建表、索引、唯一约束等数据库结构变更。 |
| Tests | `test/server/*` | Repository / Service / Handler 分层测试样板。 |

## user 模块当前示例

`user` 模块拆成两个存储模型：

- `users`：账号身份信息，包含 `user_id`、`email`、`password`。
- `user_profiles`：用户资料信息，包含 `user_id`、`nickname`。

注册流程演示了一个真实业务里常见的事务边界：

1. Service 先检查邮箱是否已存在。
2. 生成用户 ID，哈希密码。
3. 在同一个事务里创建 `users` 账号记录和 `user_profiles` 资料记录。
4. 任一 Repository 返回错误时，事务整体回滚。

更新资料流程演示了跨表更新：

1. 根据 `user_id` 读取账号和资料。
2. 如果邮箱发生变化，先检查邮箱唯一性。
3. 在同一个事务里更新账号邮箱和资料昵称。

## 新增模块推荐步骤

假设新增 `article` 模块：

1. 在 `internal/model` 新增 `Article`。
2. 在 `migrations` 新增建表迁移，并在 `migrations/all.go` 注册版本号。
3. 在 `internal/repository` 定义 `ArticleRepository` 接口和实现。
4. 在 `internal/types` 定义 Service 输入/输出，例如 `CreateArticleInput`。
5. 在 `internal/service` 定义 `ArticleService`，把事务边界放在 Service 层。
6. 在 `api/v1` 定义 HTTP DTO。
7. 在 `internal/handler` 绑定请求并调用 Service。
8. 在 `internal/router` 注册公开路由或受保护路由。
9. 在 `internal/bootstrap/module.go` 把新的 Repository / Service / Handler provider 加入 Fx module。
10. 更新 mocks，并补充 Repository / Service 测试。

## 约定

- Handler 不直接访问 Repository。
- Repository 不依赖 Handler / Service / API DTO。
- Service 不接收 `api/v1` DTO，只接收 `internal/types`。
- 事务入口放在 Service 层，通过 `repository.Transaction` 组合多个 Repository。
- 跨表写入必须尽量放在一个事务里，避免只写入一半数据。
- 对外错误优先使用 `pkg/errcode`，底层数据库错误不要直接泄漏给 API 调用方。
- 错误码和响应 envelope 参考 `docs/architecture/error.md`。
- 生成文件可以提交，但生成流程必须可重复，后续统一收敛到 `make generate`。
