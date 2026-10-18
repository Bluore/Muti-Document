# GetGrantLink 生成文档授权链接（HTTP）

文档归属者为自己的文档生成（或复用）一条**授权链接**。本接口只负责“拿链接”，不替
任何人授权；被授权用户打开链接时，才由 link 服务完成授权并返回编辑页地址。

> 授权链接与 `(document_id, 归属者)` 绑定，同一文档重复调用会复用已有链接。
> 链接数据落在 `grant_document_links` 表，默认有效期由 link-rpc 维护。
>
> 打开授权链接的完整流程与 link-api 接口见 [`docs/link/linkApi.md`](../link/linkApi.md)。

## 1. 接口

| 项 | 值 |
| --- | --- |
| 协议 | HTTP / JSON |
| 方法 | `GET` |
| 路径 | `/api/doc/v1/document/grant` |
| 鉴权 | 需要，`Authorization: Bearer <jwt>` |
| Query | `document_id`（必填，目标文档 ID） |

- 直连文档服务：`http://localhost:1101/api/doc/v1/document/grant?document_id=doc-1001`
- 经 nginx 网关：`http://localhost:8888/api/doc/v1/document/grant?document_id=doc-1001`

调用方必须是文档归属者。服务端处理链路：

```
document-api ──(1) 校验归属──▶ document-rpc
             ──(2) 生成链接──▶ link-rpc ──▶ grant_document_links
```

## 2. 请求

### 2.1 请求头

| 头 | 必填 | 说明 |
| --- | --- | --- |
| `Authorization` | 是 | `Bearer <jwt>`，当前用户 ID 由服务端从 JWT 解析 |

### 2.2 Query 参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `document_id` | string | 是 | 目标文档 ID。缺失时返回字段验证错误 |

```bash
curl -X GET "http://localhost:8888/api/doc/v1/document/grant?document_id=doc-1001" \
  -H "Authorization: Bearer <jwt>"
```

## 3. 响应

所有响应统一包裹为 `{"code":0,"msg":"","data":{...}}`，`code` 为 `0` 表示成功。

### 3.1 成功

`data.link` 是**授权中间页**链接（前端路由），形态为 `<BaseUrl>/grant/<grantId>`：

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "link": "http://localhost:8888/grant/8b1c2d3e4f5a"
  }
}
```

### 3.2 失败

| HTTP | code | msg | 场景 |
| --- | --- | --- | --- |
| 401 | 1004002 | 用户鉴权失败 | JWT 缺失 / 无效 / 过期 |
| 200 | 1004001 | 字段验证错误 | `document_id` 缺失 |
| 200 | 1024002 | 文档不存在 | `document_id` 对应文档不存在 |
| 200 | 1024003 | 没有权限操作该文档 | 当前用户不是文档归属者 |
| 500 | 1005001 | 服务器内部出错 | 调用 link-rpc 失败等 |

错误体形如：

```json
{
  "code": 1024003,
  "msg": "没有权限操作该文档",
  "data": {}
}
```

## 4. 生成规则

- **归属校验**：document-api 先调 document-rpc 取文档元信息，`owner_id` 必须等于 JWT 用户。
- **复用**：link-rpc 按 `(document_id, user_id)` 查 `grant_document_links`，命中即复用，否则新建。
- **有效期**：本接口不传过期时间，link-rpc 使用默认有效期 **7 天**（自创建时刻起）。
- **链接形态**：`<BaseUrl>/grant/<grantId>`（前端授权中间页，由 `globalkey.UrlGrant` 拼接），
  其中 `BaseUrl` 由 link-rpc 配置提供（`link-rpc/etc/link.yaml` 的 `BaseUrl`）。

## 5. 使用流程

1. 归属者登录，带 JWT 调本接口拿到 `link`。
2. 把 `link` 分享给被授权用户。
3. 被授权用户登录后打开 `link`，进入前端授权中间页 `/grant/:id`。
4. 中间页调用 `GET /api/link/v1/grant/:id`（需带 JWT）；link-api 解析授权链接，
   若该用户尚无权限则先授予文档权限，随后返回文档编辑页地址（`data.link`）。
5. 中间页拿到 `data.link` 后跳转到编辑页；失败时按返回的 `code` 自行处理
   （如跳转登录页或错误页）。

> 本接口不会替归属者做任何授权动作，只返回链接；真正的授权发生在步骤 4。

## 6. 完整示例

假设 Alice(`u-1001`) 是文档 `doc-1001` 的归属者。

### 6.1 请求

```
GET /api/doc/v1/document/grant?document_id=doc-1001 HTTP/1.1
Host: localhost:8888
Authorization: Bearer <alice-jwt>
```

### 6.2 成功响应

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "link": "http://localhost:8888/grant/8b1c2d3e4f5a"
  }
}
```

### 6.3 重复调用

再次调用时，因 `(doc-1001, u-1001)` 的链接已存在，服务端直接复用并返回同一个 `link`。

### 6.4 分享给 Bob

Bob(`u-1002`) 登录后打开该 `link`，进入前端授权中间页；中间页调用
`GET /api/link/v1/grant/8b1c2d3e4f5a`，link-api 为其授予 `doc-1001` 的编辑权限并返回：

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "link": "http://localhost:8888/documents/doc-1001/edit"
  }
}
```

前端拿到 `data.link` 后自行跳转到编辑页。

## 7. 错误场景

| 场景 | 表现 |
| --- | --- |
| 未携带 / 无效 JWT | 401 `{"code":1004002,"msg":"用户鉴权失败"}` |
| 缺少 `document_id` | 200 `{"code":1004001,"msg":"字段验证错误"}` |
| 文档不存在 | 200 `{"code":1024002,"msg":"文档不存在"}` |
| 非归属者调用 | 200 `{"code":1024003,"msg":"没有权限操作该文档"}` |
| link-rpc 不可用 | 500 `{"code":1005001,"msg":"服务器内部出错"}` |
| 打开授权链接失败 | 200 `{"code":1034001,"msg":"链接不存在"}` / `1034002` 链接已过期 / `1034003` 没有权限访问该链接 |

> 注意：复用已有链接时**不校验其过期时间**。若旧链接已过期，本接口仍会原样返回，
> 但用户打开时会被 link-api 判定为过期并返回 `1034002`。
