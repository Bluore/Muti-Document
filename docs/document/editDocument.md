# EditDocument 协同编辑（WebSocket）

文档实时协同编辑接口。客户端与服务端建立 WebSocket 长连接；同一文档的所有在线用户
共享一份进程内的 OT 状态（`connection.DocumentClient`，按 `document_id` 隔离）。

> 该接口由独立的 `edit-api` 服务提供（`app/document/cmd/edit`）。编辑内容、光标与**文档标题**
> 的持久化仍以进程内 OT 状态为主（变更缓存于 redis 并异步落库），因此该服务暂不支持横向扩容
> 或多实例部署。
>
> 该服务在同一进程内另提供 gRPC `edit.rpc`（`pb/edit.proto`）的 `StopDocumentSession`，
> 供其他服务关闭某个文档的内存会话；`document-api` 删除文档时会调用它。

## 1. 接口

| 项 | 值 |
| --- | --- |
| 协议 | WebSocket（`ws://` / `wss://`），业务消息为 JSON 文本帧 |
| 方法 | `GET`（HTTP 升级握手） |
| 路径 | `/api/edit/v1/document` |
| Query | `document_id`（必填，目标文档 ID） |

- 直连编辑服务：`ws://localhost:1101/api/edit/v1/document?document_id=doc-1001`
- 经 nginx 网关：`ws://localhost:8888/api/edit/v1/document?document_id=doc-1001`

鉴权**不在 HTTP 层完成**（该路由未挂 `JwtAuth`）。连接建立后，客户端必须把 JWT 作为
**第一条 WebSocket 文本消息**发送：

```json
{"token": "<jwt>"}
```

握手流程：

1. 客户端发起 WebSocket 升级，`document_id` 放在 query。
2. 客户端发送首条消息 `{"token":"..."}`。
3. 服务端校验 JWT，经 user-rpc 查询用户信息，取出（或创建）该文档的 OT 状态。
4. 服务端在同一把锁下取出一致快照，下发 `init` 消息（含当前全文、版本号、标题、在线用户
   及其光标快照），并向房间内**其他**用户广播 `user.join_users`（新用户本身不接收自己的
   join）。
5. 进入双向消息循环，服务端按需广播 `edit` / `cursor` / `user` / `set_title` 消息。
6. 任一端关闭连接后，服务端向**剩余**用户广播 `user.quit_users`。

## 2. 消息总览

- 客户端 → 服务端：`ClientMessage`（`edit` / `cursor` / `set_title`）
- 服务端 → 客户端：`ServerMessage`（`init` / `edit` / `cursor` / `user` / `set_title`）

`ServerMessage` 的五个字段**始终存在**，未携带的为 `null`；`ClientMessage` 的三个字段均带
`omitempty`，为空时会被省略。

## 3. 请求消息（客户端 → 服务端）

### 3.1 ClientMessage

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `edit` | ClientEditMsg \| null | 一次文本操作（可选） |
| `cursor` | Cursor \| null | 光标 / 选区位置（可选） |
| `set_title` | ClientSetTitleMsg \| null | 修改文档标题（可选） |

三个字段可任意组合：可同时发送，也可只发其一。

### 3.2 ClientEditMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `reversion` | uint32 | **必填**。本次操作所基于的版本号（取自上一次收到的 `init` / `edit`） |
| `operator` | array | **必填**。操作序列，见 3.5 |

`operator` 是**裸数组**，不是对象。原因：服务端收到后会自行补上操作者 `id`（以及仅供内部
OT 变换使用的 `reversion`），客户端无需（也无法）提供这些字段。

```json
{
  "edit": { "reversion": 0, "operator": [5, " ", 5] },
  "cursor": null
}
```

### 3.3 Cursor

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | string | 服务端会覆盖为当前连接用户，可留空 |
| `reversion` | uint32 | 该光标所基于的版本号 |
| `anchor` | { offset, assoc } | 选区起点（普通光标时 anchor == head） |
| `head` | { offset, assoc } | 选区终点 |

- `offset`：以**字节**为单位的偏移（与 `Operator` 同一单位）；负数会被钳制为 0。
- `assoc`（association）：当其他用户恰好在该位置插入文本时，光标端点的归属：
  - `assoc < 0`：粘在插入点左侧，光标不动；
  - `assoc >= 0`：粘在插入点右侧，光标被推到插入文本之后。

```json
{
  "cursor": {
    "user_id": "",
    "reversion": 1,
    "anchor": { "offset": 6, "assoc": 1 },
    "head":   { "offset": 6, "assoc": 1 }
  }
}
```

### 3.4 ClientSetTitleMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `title` | string | **必填**（可为空串）。文档新标题 |

- 设置者身份由服务端从当前连接取得，客户端**无需**（也无法）提供 `user_id`。
- 标题没有版本号，不参与 OT；服务端直接覆盖为最新值并广播（见 4.5）。
- 若新标题与当前标题相同，服务端视为空操作，不广播。

```json
{
  "set_title": { "title": "2026 Q3 周报" }
}
```

### 3.5 operator 的编码

`operator` 数组按顺序描述“保留 / 插入 / 删除”，元素含义由 JSON 值的类型决定：

| JSON 值 | 含义 |
| --- | --- |
| 非负整数 `n` | `Retain(n)`：跳过 n 个字符 |
| 字符串 `"s"` | `Insert("s")`：插入字符串 s |
| 负整数 `-n` | `Delete(n)`：删除 n 个字符 |

例：在 `"helloworld"` 的第 5 个字符处插入一个空格：

```
[5, " ", 5]      // 等价于 Retain 5, Insert " ", Retain 5
```

## 4. 响应消息（服务端 → 客户端）

### 4.1 ServerMessage

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `init` | ServerInitMsg \| null | 初始化快照，连接建立后仅下发一次 |
| `edit` | ServerEditMsg \| null | 一批已应用的操作 |
| `cursor` | Cursor[] \| null | 一批光标更新（已按用户去重，取最新） |
| `user` | ServerUserMsg \| null | 用户加入 / 离开通知，连接建立与断开时广播 |
| `set_title` | ServerSetTitleMsg \| null | 文档标题变更通知 |

一条 `ServerMessage` 可以**同时**携带 `edit` 与 `cursor`；`user`、`set_title` 通常单独出现，
但一小段时间内堆积的多条广播在写回 socket 前会被合并（见 4.4 / 4.5），因此也可能与其他
字段共存。

### 4.2 ServerInitMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `reversion` | uint32 | 当前版本号（已应用操作数） |
| `title` | string | 当前文档标题；从未设置过时为空串 `""` |
| `content` | string | 当前文档全文 |
| `user_id` | string | 当前连接用户 ID |
| `users` | User[] | 当前在线用户（含自己），顺序不保证 |
| `cursor` | Cursor[] | 当前在线用户的光标快照（含自己），顺序不保证 |
| `cursor_color` | string | 服务端为当前用户随机分配的光标颜色 |

> `reversion`、`title`、`content`、`users`、`cursor` 由服务端在**同一把锁**下一次性取出，
> 是一份相互一致的快照。

`User`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 用户 ID |
| `name` | string | 昵称 |
| `cursor_color` | string | 光标颜色（12 色调色板之一） |

### 4.3 ServerEditMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `reversion` | uint32 | 应用这批操作后的**新版本号** |
| `operator` | Operator[] | 一批操作对象 |

与请求不同，服务端下发的 `operator` 是**对象数组**，每项形如：

```json
{ "id": "u-1001", "ops": [5, " ", 5] }
```

> 注意收发格式不对称：客户端**发送**用裸数组 `[5, " ", 5]`，服务端**下发**用对象数组。
> 该对象由 `Operator.MarshalJSON` 经 `map[string]any` 输出，`encoding/json` 按键名**字典序**
> 排列，因此实际字段顺序是 `id`、`ops`。
>
> 下发的 operator **不含** `reversion`（它由服务端在内部用于 OT 变换，不对外暴露）；这批操作
> 对应的版本号由外层 `ServerEditMsg.reversion` 给出。

### 4.4 ServerUserMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `join_users` | User[] \| null | 本次加入的用户 |
| `quit_users` | User[] \| null | 本次离开的用户 |

一条 `user` 消息只会携带其中一个字段，另一个为 `null`。触发时机：

- 新用户连接成功时，向房间内**其他**用户广播 `join_users`；新用户本身不接收自己的
  join（它已通过 `init.users` 获知完整名册）。
- 用户断开连接时，向**剩余**用户广播 `quit_users`；离开者已从订阅列表移除，不接收自己的
  quit。

同一条 `user` 消息在写回 socket 前可能被合并：一小段时间内堆积的多个 join / quit 会折叠
成一条，按用户 `id` 去重（后到的覆盖先到的），且同一用户同时出现在 join 与 quit 时以
quit 为准。

### 4.5 ServerSetTitleMsg

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | string | 发起本次标题变更的用户 ID |
| `title` | string | 变更后的文档标题 |

触发时机与语义：

- 任一客户端发送 `set_title` 后，服务端更新标题并向**所有**连接广播（**包括设置者自己**），
  使所有客户端收敛到同一标题。
- 若新标题与当前标题相同，不广播。
- 标题变更不带版本号、不参与 OT，后到的变更直接覆盖先到的。
- 与 `user` 一样，短时间堆积的多条 `set_title` 在写回 socket 前会合并，取**最后一条**。

```json
{
  "init": null,
  "edit": null,
  "user": null,
  "cursor": null,
  "set_title": { "user_id": "u-1001", "title": "2026 Q3 周报" }
}
```

## 5. 心跳与控制帧

除 JSON 文本帧外，服务端还会：

- 每 **54s** 发送一个 WebSocket `Ping` 控制帧，并要求客户端在 **60s** 内回 `Pong`；
  超时即视为掉线并关闭连接。客户端需实现标准 `Pong` 响应（多数 WebSocket 库自动处理）。
- 当同一用户重复连接同一文档时，发送关闭帧 `1008 (Policy Violation)`，原因文本为
  `user already connected`。

## 6. 示例

假设文档 `doc-1001` 初始内容为 `"helloworld"`、初始标题为空串，用户 Alice(`u-1001`) 与
Bob(`u-1002`) 依次接入。`cursor_color` 为随机值，示例中假设 Alice 为 `#64B5F6`、
Bob 为 `#FFB74D`。

### 6.1 Alice 建立连接并收到 init

客户端发送：

```json
{"token": "<alice-jwt>"}
```

服务端返回：

```json
{
  "init": {
    "reversion": 0,
    "title": "",
    "content": "helloworld",
    "user_id": "u-1001",
    "users": [
      { "id": "u-1001", "name": "Alice", "cursor_color": "#64B5F6" }
    ],
    "cursor": [],
    "cursor_color": "#64B5F6"
  },
  "edit": null,
  "cursor": null,
  "user": null,
  "set_title": null
}
```

### 6.2 Alice 在偏移 5 处插入空格

当前 `helloworld` → 目标 `hello world`。客户端发送：

```json
{
  "edit": { "reversion": 0, "operator": [5, " ", 5] },
  "cursor": null
}
```

服务端应用后内容变为 `hello world`、版本号变为 1，并向**所有在线连接（含 Alice 自己）**
广播：

```json
{
  "edit": {
    "reversion": 1,
    "operator": [
      { "id": "u-1001", "ops": [5, " ", 5] }
    ]
  },
  "init": null,
  "cursor": null,
  "user": null,
  "set_title": null
}
```

### 6.3 Bob 接入并上报光标

Bob 发送 `{"token":"<bob-jwt>"}`，收到 init（此时版本 1、内容 `hello world`、标题仍为空）：

```json
{
  "init": {
    "reversion": 1,
    "title": "",
    "content": "hello world",
    "user_id": "u-1002",
    "users": [
      { "id": "u-1001", "name": "Alice", "cursor_color": "#64B5F6" },
      { "id": "u-1002", "name": "Bob", "cursor_color": "#FFB74D" }
    ],
    "cursor": [],
    "cursor_color": "#FFB74D"
  },
  "edit": null,
  "cursor": null,
  "user": null,
  "set_title": null
}
```

与此同时，Alice（房间内其他用户）会收到 Bob 加入的 `user` 通知：

```json
{
  "edit": null,
  "init": null,
  "user": {
    "join_users": [
      { "id": "u-1002", "name": "Bob", "cursor_color": "#FFB74D" }
    ],
    "quit_users": null
  },
  "cursor": null,
  "set_title": null
}
```

Bob 把光标移到偏移 6（`hello ` 之后），发送：

```json
{
  "cursor": {
    "user_id": "",
    "reversion": 1,
    "anchor": { "offset": 6, "assoc": 1 },
    "head":   { "offset": 6, "assoc": 1 }
  }
}
```

服务端向所有连接广播：

```json
{
  "init": null,
  "edit": null,
  "user": null,
  "cursor": [
    {
      "user_id": "u-1002",
      "reversion": 1,
      "anchor": { "offset": 6, "assoc": 1 },
      "head":   { "offset": 6, "assoc": 1 }
    }
  ],
  "set_title": null
}
```

### 6.4 Alice 再次编辑，Bob 的光标被自动平移

Alice 在偏移 6 处插入 `big `（目标 `hello big world`）。发送：

```json
{
  "edit": { "reversion": 1, "operator": [6, "big ", 5] },
  "cursor": null
}
```

服务端应用后版本变为 2，并把 Bob 的光标（偏移 6，`assoc=1`）随插入向右平移 4 位到偏移
10。广播同时携带操作与平移后的光标：

```json
{
  "edit": {
    "reversion": 2,
    "operator": [
      { "id": "u-1001", "ops": [6, "big ", 5] }
    ]
  },
  "init": null,
  "user": null,
  "cursor": [
    {
      "user_id": "u-1002",
      "reversion": 1,
      "anchor": { "offset": 10, "assoc": 1 },
      "head":   { "offset": 10, "assoc": 1 }
    }
  ],
  "set_title": null
}
```

> `cursor.reversion` 表示该客户端**上报光标时**所处的版本；服务端在随后的编辑中只平移
> 偏移、不改写 `reversion`，因此它可能小于当前文档版本。

### 6.5 Alice 修改文档标题

Alice 发送：

```json
{
  "set_title": { "title": "2026 Q3 周报" }
}
```

服务端记录标题后，向**所有**连接（含 Alice 自己）广播：

```json
{
  "init": null,
  "edit": null,
  "user": null,
  "cursor": null,
  "set_title": { "user_id": "u-1001", "title": "2026 Q3 周报" }
}
```

此后新接入的用户会在 `init.title` 中直接拿到 `"2026 Q3 周报"`。若 Alice 再发一次相同标题，
服务端不会广播。

### 6.6 Bob 断开连接

Bob 关闭连接后，Alice（仍在房间内的用户）会收到离开通知：

```json
{
  "edit": null,
  "init": null,
  "user": {
    "join_users": null,
    "quit_users": [
      { "id": "u-1002", "name": "Bob", "cursor_color": "#FFB74D" }
    ]
  },
  "cursor": null,
  "set_title": null
}
```

## 7. 错误与关闭

| 场景 | 表现 |
| --- | --- |
| 缺少 `document_id` | 升级前返回 HTTP JSON：`{"code":1004001,...}`（字段验证错误） |
| 首帧非法或 token 无效 | 关闭连接（握手已完成，不会收到 JSON 错误体） |
| 用户不存在 | 关闭连接 |
| 同一用户重复连接同一文档 | 关闭帧 `1008`，原因 `user already connected` |
| 心跳超时（60s 无 Pong） | 关闭连接 |
| 操作的 `reversion` 大于服务端当前版本 | 服务端忽略该操作并记日志，不广播 |
| `set_title.title` 与当前标题相同 | 视为空操作，不广播、不报错 |
