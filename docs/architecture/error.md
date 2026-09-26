# 错误码与响应约定

pi-layout 使用 `pkg/errcode` 描述业务错误。`errcode.Error` 实现了 Pi 的 `Code()` 和 `StatusCode()` 接口，因此 Handler / Service 返回该错误时，Pi 会自动渲染统一响应。

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

业务错误默认返回 HTTP `400 Bad Request`，但响应体中的 `code` 保留业务码，例如 `1001`。

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

## 未来对齐方向

gRPC 目前仍直接返回 error。后续如果 Pi 框架提供 unified error model，layout 的 `pkg/errcode` 应作为适配层对齐 HTTP / gRPC / CLI 的错误语义。
