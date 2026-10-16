#!/usr/bin/env sh
# 初始化 Qdrant: 创建 document_chunks 集合与 payload 索引, 可重复执行
set -eu

DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

QDRANT_URL=${QDRANT_URL:-http://localhost:6333}
COLLECTION=${COLLECTION:-document_chunks}
RETRIES=${RETRIES:-30}

say() { printf '%s\n' "[qdrant-init] $*"; }

# 等待 Qdrant 就绪
i=0
while [ "$i" -lt "$RETRIES" ]; do
	if curl -fsS -o /dev/null "$QDRANT_URL/readyz"; then
		break
	fi
	i=$((i + 1))
	say "等待 Qdrant 就绪 ($i/$RETRIES) ..."
	sleep 2
done
if [ "$i" -ge "$RETRIES" ]; then
	say "Qdrant 未就绪: $QDRANT_URL" >&2
	exit 1
fi

# 创建集合(已存在则跳过, 不覆盖数据)
code=$(curl -s -o /dev/null -w '%{http_code}' "$QDRANT_URL/collections/$COLLECTION")
if [ "$code" = "200" ]; then
	say "集合 $COLLECTION 已存在, 跳过创建"
else
	say "创建集合 $COLLECTION"
	curl -fsS -X PUT "$QDRANT_URL/collections/$COLLECTION" \
		-H 'Content-Type: application/json' \
		--data-binary "@$DIR/document_chunks.json" >/dev/null
fi

# payload 索引(doc_id/owner_id/chunk_index/create_at/text)
say "创建 payload 索引"
index() {
	curl -fsS -X PUT "$QDRANT_URL/collections/$COLLECTION/index?wait=true" \
		-H 'Content-Type: application/json' \
		-d "$1" >/dev/null
}
index '{"field_name":"doc_id","field_schema":"keyword"}'
index '{"field_name":"owner_id","field_schema":"keyword"}'
index '{"field_name":"chunk_index","field_schema":"integer"}'
index '{"field_name":"create_at","field_schema":"datetime"}'
index '{"field_name":"text","field_schema":{"type":"text","tokenizer":"multilingual","min_token_len":2,"lowercase":true}}'

info=$(curl -fsS "$QDRANT_URL/collections/$COLLECTION")
points=$(printf '%s' "$info" | tr -d ' ' | sed -n 's/.*"points_count":\([0-9]*\).*/\1/p')
say "完成: $QDRANT_URL/collections/$COLLECTION (points=${points:-0})"
