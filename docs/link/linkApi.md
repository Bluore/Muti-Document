# Link 服务 API（授权链接 / 短链）

授权链接流程涉及以下入口：

| 入口 | 服务 | 说明 |
| --- | --- | --- |
| `/api/doc/v1/document/grant` | document-api | 归属者生成（或复用）授权链接 |
| `/api/link/v1/grant/{id}` | link-api | 打开授权链接：授予权限并换取文档地址 |
| `/s/{id}` | link-api | 访问短链（302 跳转，非 JSON） |

> 网关地址示例：`http://localhost:8888`（nginx `8081` → 宿主机 `8888`）。
> 各服务容器内均监听 `1101`。返回的链接前缀由 link-rpc 配置的 `BaseUrl` 决定
> （`app/link/cmd/rpc/etc/link.yaml`）。

## 1. 生成授权链接（归属者）

```
GET /api/doc/v1/document/grant?document_id={documentId}
```

| 项 | 值 |
| --- | --- |
| 鉴权 | 需要，`Authorization: Bearer <jwt>` |
| Query | `document_id`：目标文档 ID，必填 |
| 权限 | 仅文档归属者可调用 |

**成功响应**

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "link": "http://localhost:8888/grant/8b1c2d3e4f5a"
  }
}
```

`data.link` 是**前端授权中间页**地址，形态为 `<BaseUrl>/grant/<grantId>`。

**失败响应**

| HTTP | code | msg | 场景 |
| --- | --- | --- | --- |
| 401 | 1004002 | 用户鉴权失败 | JWT 缺失 / 无效 / 过期 |
| 200 | 1004001 | 字段验证错误 | `document_id` 缺失 |
| 200 | 1024002 | 文档不存在 | 文档不存在 |
| 200 | 1024003 | 没有权限操作该文档 | 当前用户不是归属者 |
| 500 | 1005001 | 服务器内部出错 | 调用 link-rpc 失败等 |

**示例**

```bash
curl "http://localhost:8888/api/doc/v1/document/grant?document_id=doc-1001" \
  -H "Authorization: Bearer <jwt>"
```

**规则**

- 链接与 `(document_id, 归属者)` 绑定，重复调用复用同一条链接。
- 默认有效期 **7 天**，本接口不传过期时间。
- 复用时不校验剩余有效期。

## 2. 授权中间页（前端路由）

```
GET /grant/{grantId}
```

这是前端页面，不是后端接口（nginx 会回落到前端）。页面职责：

1. 获取当前登录态；
2. 调用下文 `GET /api/link/v1/grant/{grantId}` 换取文档地址；
3. 按返回结果跳转到 `data.link`，或按错误码跳转登录页 / 错误页。

## 3. 换取文档地址（link-api）

```
GET /api/link/v1/grant/{grantId}
```

| 项 | 值 |
| --- | --- |
| 鉴权 | 需要，`Authorization: Bearer <jwt>` |
| Path | `grantId`：授权链接 ID |
| 行为 | 校验链接 → 检查文档权限 → 无权限则授权 → 返回文档编辑页地址 |

**成功响应（HTTP 200）**

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "link": "http://localhost:8888/documents/doc-1001/edit"
  }
}
```

`data.link` 即最终目标地址，形态为 `<BaseUrl>/documents/<documentId>/edit`。
接口**不再返回 302**，由前端读取 `data.link` 后自行跳转。

**失败响应**

| HTTP | code | msg | 场景 |
| --- | --- | --- | --- |
| 401 | 1004002 | 用户鉴权失败 | JWT 缺失 / 无效 / 过期 |
| 200 | 1034001 | 链接不存在 | `grantId` 不存在，或对应文档已删除 |
| 200 | 1034002 | 链接已过期 | 超过有效期 |
| 200 | 1034003 | 没有权限访问该链接 | 授予文档权限失败 |
| 500 | 1005001 | 服务器内部出错 | 下游 RPC 调用失败等 |

错误体形如：

```json
{
  "code": 1034002,
  "msg": "链接已过期",
  "data": {}
}
```

**示例**

```bash
curl "http://localhost:8888/api/link/v1/grant/8b1c2d3e4f5a" \
  -H "Authorization: Bearer <jwt>"
```

## 4. 访问短链（公开）

```
GET /s/{id}
```

| 项 | 值 |
| --- | --- |
| 鉴权 | 不需要 |
| 行为 | **302 跳转**到短链目标地址 |

- 成功：`302`，`Location: <targetUrl>`
- 失败（不存在 / 已过期）：`302`，`Location: /err/404`

> 短链目前只提供访问入口，创建能力由 link-rpc 的 `CreateShortLink` 提供，
> 未暴露 HTTP 接口。

## 5. 完整流程

```
归属者前端             document-api              link-rpc                 link-api
    |                       |                        |                        |
    |-- GET /api/doc/v1/document/grant?document_id -->|                        |
    |                       |-- CreateGrantLink ---->|                        |
    |<-- data.link=/grant/{grantId} -----------------|                        |
    |                                                                          |
    |  分享链接                                                                 |
    |                                                                          |
被授权用户前端打开 /grant/{grantId}（前端页面）                                 |
    |-- GET /api/link/v1/grant/{grantId} (JWT) ------------------------------>|
    |                       |                        |  校验链接/授权          |
    |<-- data.link=/documents/{documentId}/edit -----------------------------|
    |  前端跳转到 data.link                                                    |
```

## 6. 错误码汇总

| code | HTTP | msg | 所属 |
| --- | --- | --- | --- |
| 0 | 200 | 成功 | 公共 |
| 1004001 | 200 | 字段验证错误 | 公共 |
| 1004002 | 401 | 用户鉴权失败 | 公共 |
| 1005001 | 500 | 服务器内部出错 | 公共 |
| 1024002 | 200 | 文档不存在 | document-api |
| 1024003 | 200 | 没有权限操作该文档 | document-api |
| 1034001 | 200 | 链接不存在 | link-api |
| 1034002 | 200 | 链接已过期 | link-api |
| 1034003 | 200 | 没有权限访问该链接 | link-api |

## 7. 网关路由

| 路径 | 转发目标 |
| --- | --- |
| `/api/doc/` | document-api |
| `/api/link/` | link-api |
| `/s/` | link-api |
| `/grant/` | 前端（兜底 `/`） |

配置见 `deploy/nginx/conf.d/mutidoc.conf`。
