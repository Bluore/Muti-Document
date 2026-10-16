# Qdrant 初始化

文档向量库: 把 `documents` 表里的文档切块(chunk)后向量化, 供检索/问答使用。
一个 Qdrant 集合 `document_chunks`, 向量字段固定为 `chunk`(1024 维, Cosine)。

## 文件

| 文件 | 说明 |
| --- | --- |
| `document_chunks.json` | 集合定义(`PUT /collections/{name}` 请求体): 向量名/维度/距离、HNSW、优化器 |
| `init.sh` | 幂等初始化脚本: 等待就绪 → 建集合 → 建 payload 索引 |

## 数据结构

Qdrant 没有建表 SQL, 集合的"结构"由向量配置 + payload 索引表达。
每个 point 的 payload 固定为以下字段(点 ID 需自备):

| 字段 | 类型 | 索引 | 说明 |
| --- | --- | --- | --- |
| `doc_id` | string | keyword | 所属文档 ID, 对应 `documents.id` |
| `owner_id` | string | keyword | 文档归属用户, 用于按权限过滤 |
| `chunk_index` | integer | integer | 分块序号, 从 0 开始 |
| `text` | string | text (multilingual) | 该块原文, 返回给模型 |
| `create_at` | string | datetime | 分块时间, RFC3339 如 `2026-10-16T12:00:00Z` |
| `chunk` | vector(1024) | — | `text` 的向量(与 payload 平行的向量字段) |

点 ID 用 `UUID`(Qdrant 只接受 UUID 或 uint64)。建议对 `doc_id + "#" + chunk_index`
取 UUIDv5 做稳定 ID, 这样同一文档重新切块时直接覆盖而不会产生重复。

## 执行

Qdrant 由 `docker-compose.env.yml` 启动(`6333` REST / `6334` gRPC)。
初始化是一次性的, 重复执行安全(集合已存在则只补索引):

```sh
# 宿主机(或 Git Bash/WSL), Qdrant 端口已映射
QDRANT_URL=http://localhost:6333 sh deploy/qdrant/init.sh

# 容器网络内(免装 curl), 二选一
docker run --rm --network mutidoc-net -v "$PWD/deploy/qdrant:/init" curlimages/curl \
  sh -c 'QDRANT_URL=http://qdrant:6333 sh /init/init.sh'
```

验证:

```sh
curl -s http://localhost:6333/collections/document_chunks
```

## 约定

- 向量维度 **1024**: 与 `BAAI/bge-m3`(硅基流动) 输出一致。**换模型必须改
  `document_chunks.json` 的 `size`、两条 yaml 的 `Qdrant.VectorSize` 并重建集合**。
- 向量名固定 `chunk`, 写入/查询时都要显式指定(`"vector": {"name": "chunk", ...}`),
  不能用默认向量名。
- 检索时先按 `owner_id` / `doc_id` 过滤再向量搜索; 权限校验仍以 `documents` 表为准。
- 集合创建后不要手工改维度; 需要重来就删掉
  `data/qdrant/storage` 后再执行一次 `init.sh`。

## 代码侧

| 位置 | 作用 |
| --- | --- |
| `app/document/model/documentChunksModel.go` | 集合读写模型: 集合/索引懒创建、按 `doc_id` 删除或读取、写入分块、向量检索 |
| `app/document/model/documentChunks.go` | 分块结构与切块规则(定长 + 向前重叠) |
| `pkg/embedding/embedding.go` | OpenAI 兼容的 `/embeddings` 客户端 |
| `app/document/cmd/edit/internal/indexer/indexer.go` | 正文落库后异步重建索引 |
| `app/document/cmd/rpc/internal/logic/searchdocumentlogic.go` | `SearchDocument` RPC, 检索结果序列化为 JSON |

集合与 payload 索引由代码在首次读写时按同一套定义幂等创建, 因此**跳过本节的手动
初始化也能跑**, 但 `init.sh` 仍是最先建好集合的推荐方式。

服务配置(`edit.yaml` / `document.yaml`, docker 下为 `deploy/config/*.yml`):

| 配置 | 说明 |
| --- | --- |
| `Qdrant.Host/Port` | 容器内是 `qdrant:6334` (gRPC) |
| `Qdrant.Collection` | 默认 `document_chunks`, 需与本节集合名一致 |
| `Qdrant.VectorSize` | 默认 1024, 与 `document_chunks.json` 的 `size` 一致 |
| `Embedding.*` | OpenAI 兼容端点(默认硅基流动 `https://api.siliconflow.cn/v1`, `BAAI/bge-m3`); `APIKey` 留空则无法向量化 |
| `Chunk.Size/Overlap` | 切块长度与重叠字符数, 默认 500 / 100 |

RPC 用法见 `docs/document/searchDocument.md`。
