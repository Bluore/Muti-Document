# 文档快照 API（发布 / 读取）

归属者把文档**当前内容**发布成一份只读快照，拿到一条可分享的短链；任何人通过快照 ID
都能直接读取快照内容，无需登录。

> 快照写入 `public_documents` 表，与 `documents` 完全独立：之后继续编辑原文档不会影响
> 已发布的快照。每次发布都会生成一份新的快照与新的短链。

## 1. 接口一览

| 接口 | 方法 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| `/api/doc/v1/document/snapshot` | `POST` | 需要 JWT | 发布快照，返回 `snapshot_id` 与短链 |
| `/api/doc/v1/document/snapshot/content` | `GET` | 无需鉴权 | 按 `snapshot_id` 读取快照正文 |

处理链路：

```
发布：document-api ──(1) 校验归属──▶ document-rpc ──(2) 写 public_documents
                                    └──(3) 建短链──▶ link-rpc ──▶ short_links

读取：调用方 ──GET snapshot_id──▶ document-api ──查 public_documents──▶ document-rpc
```

- **鉴权与归属校验只在 API 层**：document-api 从 JWT 取用户，并确认其为文档归属者。
- **document-rpc 只负责增和查**：发布时插入快照并调用 link-rpc 建短链，读取时直接查表。

## 2. 发布文档快照

### 2.1 接口

| 项 | 值 |
| --- | --- |
| 协议 | HTTP / JSON |
| 方法 | `POST` |
| 路径 | `/api/doc/v1/document/snapshot` |
| 鉴权 | 需要，`Authorization: Bearer <jwt>` |
| Body | `document_id`（必填，目标文档 ID） |

- 直连文档服务：`http://localhost:1101/api/doc/v1/document/snapshot`
- 经 nginx 网关：`http://localhost:8888/api/doc/v1/document/snapshot`

### 2.2 请求

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `document_id` | string | 是 | 要发布的文档 ID |

```bash
curl -X POST "http://localhost:8888/api/doc/v1/document/snapshot" \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"document_id":"doc-1001"}'
```

### 2.3 成功响应

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "snapshot_id": "snapshot-7f3a9c",
    "short_link": "http://localhost:8888/s/8b1c2d"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `snapshot_id` | string | 新快照 ID，可用于读取快照内容 |
| `short_link` | string | 短链，形如 `<BaseUrl>/s/<linkId>` |

### 2.4 失败响应

| HTTP | code | msg | 场景 |
| --- | --- | --- | --- |
| 401 | 1004002 | 用户鉴权失败 | JWT 缺失 / 无效 / 过期 |
| 200 | 1004001 | 字段验证错误 | `document_id` 缺失 |
| 200 | 1024002 | 文档不存在 | 文档不存在或已删除 |
| 200 | 1024003 | 没有权限操作该文档 | 当前用户不是文档归属者 |
| 500 | 1005001 | 服务器内部出错 | 写快照或调用 link-rpc 失败 |

### 2.5 规则

- **仅归属者可发布**：document-api 先取文档元信息，`owner_id` 必须等于 JWT 用户。
- **快照内容**：取数据库正文与 redis 中尚未落库的协作变更合并后的结果，标题与版本号同步写入。
- **每次发布都是新快照**：`snapshot_id` 由 document-rpc 生成，不复用旧快照。
- **短链有效期**：本接口不传入过期时间，link-rpc 使用默认有效期 **7 天**。
- **短链目标**：`/public_documents/<snapshot_id>`（前端公开文档页，由 `globalkey.UrlPublicDocument` 拼接）。

## 3. 获取快照内容

### 3.1 接口

| 项 | 值 |
| --- | --- |
| 协议 | HTTP / JSON |
| 方法 | `GET` |
| 路径 | `/api/doc/v1/document/snapshot/content` |
| 鉴权 | **无需** |
| Query | `snapshot_id`（必填，快照 ID） |

```bash
curl "http://localhost:8888/api/doc/v1/document/snapshot/content?snapshot_id=snapshot-7f3a9c"
```

### 3.2 成功响应

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "id": "snapshot-7f3a9c",
    "owner_id": "u-1001",
    "summary": "项目设计文档",
    "content": "快照正文……",
    "version": 12,
    "create_at": 1760000000,
    "update_at": 1760000000
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 快照 ID |
| `owner_id` | string | 原文档归属者 |
| `summary` | string | 发布时的文档标题 |
| `content` | string | 发布时的文档正文 |
| `version` | int64 | 发布时的文档版本号 |
| `create_at` | int64 | 快照创建时间，Unix 秒 |
| `update_at` | int64 | 快照更新时间，Unix 秒 |

### 3.3 失败响应

| HTTP | code | msg | 场景 |
| --- | --- | --- | --- |
| 200 | 1004001 | 字段验证错误 | `snapshot_id` 缺失 |
| 200 | 1024004 | 文档快照不存在 | 快照 ID 不存在 |
| 500 | 1005001 | 服务器内部出错 | 数据库查询失败 |

## 4. 短链访问流程

1. 归属者调用发布接口，拿到 `short_link`。
2. 把短链分享给任何人；访问 `GET /s/<linkId>`（无需登录）。
3. link-api 查到短链对应的目标地址 `/public_documents/<snapshot_id>`，返回 **302** 跳转。
4. 浏览器打开前端公开文档页，页面调用 `GET /api/doc/v1/document/snapshot/content?snapshot_id=...`
   渲染快照内容。

失败时统一 302 到 `/err/404`：

| 场景 | 表现 |
| --- | --- |
| 短链不存在 | 302 → `/err/404` |
| 短链已过期 | 302 → `/err/404` |

## 5. gRPC 接口

### 5.1 发布快照

```proto
rpc PublishDocumentSnapshot(PublishDocumentSnapshotReq) returns (PublishDocumentSnapshotResp);
```

| 请求字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `DocumentId` | string | 是 | 目标文档 ID |

| 响应字段 | 类型 | 说明 |
| --- | --- | --- |
| `SnapshotId` | string | 新快照 ID |
| `ShortLink` | string | 短链 |

> RPC 不做归属校验，调用方（document-api）必须先完成鉴权与归属确认。

处理流程：

1. 按 `DocumentId` 查 `documents`（只查未删除文档），不存在返回 `DOCUMENT_NOT_EXIST_ERROR`。
2. 合并 redis 未落库变更，得到正文、标题与版本号。
3. 以新 ID 写入 `public_documents`。
4. 调用 link-rpc `CreateShortLink`，目标为 `/public_documents/<SnapshotId>`。
5. 返回快照 ID 与短链。

### 5.2 读取快照

```proto
rpc GetSnapshotDocument(GetSnapshotDocumentReq) returns (GetSnapshotDocumentResp);
```

| 请求字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `SnapshotId` | string | 是 | 快照 ID |

| 响应字段 | 类型 | 说明 |
| --- | --- | --- |
| `Id` | string | 快照 ID |
| `OwnerId` | string | 原文档归属者 |
| `Summary` | string | 快照标题 |
| `Content` | string | 快照正文 |
| `Version` | int64 | 快照版本号 |
| `CreateAt` | int64 | 创建时间，Unix 秒 |
| `UpdateAt` | int64 | 更新时间，Unix 秒 |

> RPC 不做鉴权，按快照 ID 直接返回内容；快照不存在返回
> `DOCUMENT_SNAPSHOT_NOT_EXIST_ERROR`。

### 5.3 RPC 错误码

| 错误码 | 名称 | 场景 |
| --- | --- | --- |
| `100002` | `REQUEST_PARAM_ERROR` | `DocumentId` / `SnapshotId` 为空 |
| `102001` | `DOCUMENT_NOT_EXIST_ERROR` | 原文档不存在或已删除 |
| `102003` | `DOCUMENT_SNAPSHOT_NOT_EXIST_ERROR` | 快照不存在 |
| `100001` | `SERVER_COMMON_ERROR` | 数据库读写或短链服务失败 |

## 6. 数据表

快照落在 `deploy/mysql/documents_table.sql` 的 `public_documents` 表：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | VARCHAR(50) | 快照 ID（主键） |
| `owner_id` | VARCHAR(50) | 原文档归属者 |
| `summary` | VARCHAR(255) | 快照标题 |
| `content` | TEXT | 快照正文 |
| `version` | INTEGER | 发布时的文档版本号 |
| `create_at` | DATETIME | 创建时间 |
| `update_at` | DATETIME | 更新时间 |
| `delete_at` | DATETIME | 软删除时间 |
| `del_state` | TINYINT | 软删除标记，`0` 正常、`1` 已删除 |

## 7. 完整示例

假设 Alice（`u-1001`）是文档 `doc-1001` 的归属者。

### 7.1 发布

```
POST /api/doc/v1/document/snapshot HTTP/1.1
Host: localhost:8888
Authorization: Bearer <alice-jwt>
Content-Type: application/json

{"document_id":"doc-1001"}
```

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "snapshot_id": "snapshot-7f3a9c",
    "short_link": "http://localhost:8888/s/8b1c2d"
  }
}
```

### 7.2 分享

把 `http://localhost:8888/s/8b1c2d` 发给任何人。访问后：

```
GET /s/8b1c2d HTTP/1.1
Host: localhost:8888
```

```
HTTP/1.1 302 Found
Location: /public_documents/snapshot-7f3a9c
```

### 7.3 读取

前端公开文档页调用：

```bash
curl "http://localhost:8888/api/doc/v1/document/snapshot/content?snapshot_id=snapshot-7f3a9c"
```

即可拿到快照正文，无需携带 JWT。
