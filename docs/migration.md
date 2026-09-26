# 迁移命令

框架依赖：Pi v0.3.0。自带业务迁移使用 SQLite。从项目根目录运行，先配置
`configs/.env` 的 `DB_DIALECT=sqlite` 和 `DB_NAME=storage/app.db`，确保父目录存在。

```sh
go build -o bin/migration ./cmd/migration
./bin/migration plan
./bin/migration up --target=20260206104000
./bin/migration up --dry-run
./bin/migration up --timeout=10m --lock-ttl=15m
./bin/migration status
```

无子命令时默认为 up。参数必须放在子命令之后；`--help` 不打开资源。
所有命令支持 `--target`、`--timeout`、`--config-dir`；dry-run、lock、lock-ttl 仅适用于 up。
配置优先级：进程环境 > `.APP_ENV.env`（未指定 APP_ENV 时为 `.local.env`）> `.env`。
配置读取不会修改进程环境，格式错误直接返回错误。

up 默认显式加锁，关闭必须写 `--lock=false`。租约默认 15 分钟、不续租；timeout 默认 10 分钟，
必须短于锁 TTL。用户函数忽略 context 时仍可能超出租约，不提供强制终止或 fencing 保证。
锁只能协调遵守同一协议的执行者。plan/status 没有执行锁，结果是某一时刻的状态快照。

stdout 是单条 JSON，包含 `command`、`ok`、`state_source`、`state_precise`、
`applied`、`skipped`、`pending`、`failed`、`plan`、`gaps`、`error`。失败原因是字符串，
不会把 Go error 编码为 `{}`。摘要在连接关闭后输出，清理失败也会使 `ok=false`。
日志写 stderr；解析参数/帮助阶段不输出执行摘要。二进制失败退出码为 1。

SIGINT/SIGTERM 取消执行 context。成功、用户失败、锁冲突、超时与取消都会走清理路径。
框架会尝试回滚事务和释放锁；数据库断连或非事务性 DDL 仍可能留下部分副作用，
请检查状态后处理，不能只凭退出码判断所有数据都已撤销。

迁移入口不调用 `pi.New()`，不启动服务器或后台重连循环；HTTP/task 仍使用现有 Fx 装配。
新增迁移使用 `Name` 和 `UpContext`，把 ctx 传给数据库调用。历史 `UP` 定义仍可编译。
不支持 down、checksum、history、schema diff 或自动分布式事务。
