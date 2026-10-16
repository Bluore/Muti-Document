# SearchDocument 向量检索文档内容（gRPC）

把待检索文本向量化，在 `document_chunks` 集合里做相似度检索，返回最相似的分块。
用于给智能助手（SearchAgent）提供文档检索能力。

索引来源：`edit` 服务在正文**落库成功后异步重建**——按固定长度切块（每块向前多带
重叠字符保留上下文）、向量化、写入 Qdrant；未变化的分块复用原向量，正文变短时删除
过时尾部。搜索侧只读，不修改索引。

## 1. 接口

| 项 | 值 |
| --- | --- |
| 协议 | gRPC |
| 服务 | `document.rpc`（document-rpc，`:1202`，etcd key `document.rpc`） |
| 方法 | `document.SearchDocument` |
| 鉴权 | 无（服务间调用，按 `OwnerId` / `DocumentId` 过滤） |

```
调用方 ──SearchDocument──▶ document-rpc ──(1) embedding 向量化──▶ embedding 服务
                                        ──(2) 相似检索────────▶ Qdrant document_chunks
```

## 2. 请求

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `Text` | string | 是 | 待检索文本，空白字符串返回参数错误 |
| `TopK` | int64 | 否 | 返回条数，`<= 0` 时默认 **3** |
| `DocumentId` | string | 否 | 只在该文档内检索 |
| `OwnerId` | string | 否 | 只在该用户归属的文档内检索 |

`DocumentId` / `OwnerId` 会下推为 Qdrant 的 payload 过滤条件；权限校验仍以
`documents` 表为准，本接口只做检索。

## 3. 响应

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Result` | string | 命中分块的 **JSON 数组**（无命中为 `[]`） |

数组中每项：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `doc_id` | string | 所属文档 ID |
| `owner_id` | string | 文档归属者 |
| `chunk_index` | int64 | 分块序号，从 0 开始 |
| `text` | string | 分块原文（带上一块的尾部重叠内容） |
| `create_at` | string | 分块时间，RFC3339 |
| `score` | number | 相似度（Cosine），越大越相似 |

### 3.1 示例

```json
[
  {
    "doc_id": "doc-1001",
    "owner_id": "user-9",
    "chunk_index": 2,
    "text": "……分块正文……",
    "create_at": "2026-10-16T04:20:00Z",
    "score": 0.8421
  }
]
```

## 4. 相关配置

`app/document/cmd/rpc/etc/document.yaml` 与 `app/document/cmd/edit/etc/edit.yaml`
（docker 下为 `deploy/config/*.yml`）：

| 配置 | 说明 |
| --- | --- |
| `Qdrant` | `Host/Port`（gRPC 6334）、`Collection`、`VectorSize`（默认 1024） |
| `Embedding` | OpenAI 兼容 `/embeddings`：`BaseURL`（默认硅基流动 `https://api.siliconflow.cn/v1`）/`APIKey`/`Model`（默认 `BAAI/bge-m3`）/`Dimensions`/`Timeout` |
| `Chunk` | `Size`（每块字符数）、`Overlap`（向前重叠字符数），仅 edit 使用 |

向量化默认走硅基流动的 `BAAI/bge-m3`（1024 维），需在 `Embedding.APIKey` 填入
硅基流动 API Key。`VectorSize`、`Embedding.Model` 必须与实际向量维度一致，改动后要重建
集合（见 `deploy/qdrant/README.md`）。
