# 错误码与响应约定

pi-layout 的 `pkg/errcode.Error` 是 Pi `apperror.Error` 的别名。Kind 决定协议状态，Code 保持业务身份，PublicMessage 可对外展示，WithCause 保留内部错误；HTTP/gRPC/CLI mapper 负责转换，不在业务类型中耦合协议。

## HTTP 响应 envelope

所有 JSON API 错误响应保持同一结构：

```json
{
  "code": 401,
  "data": null,
  "message": "Unauthorized"
}
```

- `code`：业务错误码。通用 HTTP 错误直接使用 HTTP status，例如 `401`。
- `message`：可安全返回给调用方的错误说明。
- `data`：错误时固定为 `null`。

## 错误码范围

| 范围 | 含义 | 示例 |
| --- | --- | --- |
| `0` | 成功 | `ErrSuccess` |
| `400-599` | 通用 HTTP / 系统错误 | `ErrBadRequest`、`ErrUnauthorized`、`ErrNotFound`、`ErrInternalServerError` |
| `1000-1999` | 业务错误 | `ErrEmailAlreadyUse`、`ErrInvalidSignature` |

业务错误按 Kind 映射状态，例如邮箱重复为 Conflict：HTTP 409、gRPC AlreadyExists，业务码 1001。
校验错误为 InvalidArgument，可携带公开的 `details: [{"field":"email","rule":"valid email required"}]`，不携带输入值。
errors.Is/As 与 errors.Join 可保留 cause；取消优先分类，其他分类取确定的错误分支，未知错误不返回原始消息。

## 分层规则

- Repository 返回底层存储错误或明确的 not found 错误，不负责写 HTTP 响应。
- Service 将业务场景映射为 `pkg/errcode`，例如邮箱重复返回 `ErrEmailAlreadyUse`。
- Handler 只做参数绑定和 DTO 转换，业务错误直接返回给 Pi responder。
- `net/http` middleware 不能直接返回 error，应使用 `errcode.WriteHTTPError`，避免手写 JSON envelope。
- 未知错误不应直接暴露给调用方；`errcode.AsError` 会把未知错误转为 `ErrInternalServerError`。

## middleware 示例

```go
if token == "" {
    errcode.WriteHTTPError(w, r, errcode.ErrUnauthorized)
    return
}
```

不要在 middleware 中复制如下响应：

```go
json.NewEncoder(w).Encode(map[string]any{
    "code": 401,
    "data": nil,
    "message": "Unauthorized",
})
```

## 跨协议映射

gRPC wrapper 与 interceptor 调用 `grpc.MapError`，通过 ErrorInfo 携带 Kind/业务码、BadRequest 携带字段规则。
CLI 可使用 `cmd.MapError` 得到安全输出和退出码；内部日志可以保留 cause，公开消息不要拼接 err.Error()。
注册/登录/读取/修改成功码分别是 201/200/200/204；204 无正文。
