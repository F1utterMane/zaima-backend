# "在吗" APP - WebSocket 协议与聊天机制

**Endpoint**: `ws://localhost:8080/api/v1/ws?token=<JWT_TOKEN>`

---

## 1. 连接与保活机制

### 鉴权
与 REST API 不同，WebSocket 的鉴权无法通过 Header (`Authorization: Bearer <token>`) 稳定跨平台传递。因此，Token 统一在握手阶段通过 `URL Query` (例如 `?token=eyJhb...`) 提供。连接建立后，服务端会在内存 Hub 中绑定该长连接与 `UserID` 的关系。

### 心跳保活 (Ping/Pong)
- **双向保活**：由于移动端网络切换频繁或容易断流，服务端设定如果 60 秒内未收到心跳或消息，将立刻断开 TCP 连接以回收服务器资源。
- **客户端行为**：客户端有责任每 `30秒` 发送一个 `Ping` 帧给服务器证明自己存活，也可以通过发送正常的业务消息来顺带刷新倒计时。

---

## 2. 消息协议与 JSON 结构

所有通过 WebSocket 传送的内容必须是符合以下 Schema 的合法 JSON 字符串。

```json
{
  "type": "chat",          // 核心控制字段, 常见值: chat / ack / system / match_ended
  "sender_id": 12,         // 发送方 UserID
  "receiver_id": 34,       // 接收方 UserID
  "msg_type": "text",      // chat 的内容类型: text / voice / image / care
  "content": "吃饭了吗？",  // 具体文字，或 OSS 上的媒体文件 URL
  "temp_id": "cli_uuid",   // 客户端生成的本地随机流水号 (用于本地去重和等待发送成功回执)
  "message_id": 99201,     // 服务端成功入库后下发的全局唯一单调递增 ID
  "timestamp": 1782334000  // 时间戳 (毫秒级)
}
```

---

## 3. 标准通讯流程图

### 3.1 发送一条普通聊天消息

1. **A (客户端)**：将语音传到 OSS 获取 URL，组装包含 `temp_id="A123"` 的 JSON，`type="chat"`, 通过 `conn.send()` 发给服务器。
2. **Server (服务端)**：
   - 收到消息，异步写入 PostgreSQL `chat_messages` 表并取得 `message_id (e.g. 500)`。
   - 检查 `B (接收者)` 是否在本地 Hub 内存中。
   - 【如果 B 在线】：利用 B 的长连接通道直接推送完整的 JSON（填上 `message_id=500`）。
   - 【如果 B 不在线】：触发离线推送（极光厂商通道）以弹出类似系统提醒的横幅通知。
3. **Server (服务端)**：
   - 给 `A (发送者)` 发送确认收到回执（ACK），内容如：
   ```json
   {
     "type": "ack",
     "temp_id": "A123",
     "message_id": 500,
     "timestamp": 1782334001
   }
   ```
4. **A (客户端)**：
   - 收到 `ACK`，将本地正在转圈发送的 `temp_id="A123"` 消息标记为发送成功 / 已读对勾。

### 3.2 接收反馈（已读未读）

1. **B (客户端)**：收到服务端推送的聊天消息，并呈现在界面上被用户阅读后。
2. **B (客户端)**：发出已读确认，将该条消息归档为已读。
   ```json
   {
     "type": "ack",
     "message_id": 500
   }
   ```
3. **Server (服务端)**：收到包含 `message_id` 的 ACK 帧后，立刻修改该条消息在 DB 中的 `is_read=true`。

---

## 4. 断网重连与补偿 (Resync)

当用户从地铁、车库出来网络恢复，重新建立 WebSocket 连接后，如何确保中间没有漏读消息？

1. 客户端在成功重连后，向 REST API 发起请求：
   `GET /api/v1/chat/history?peer_id=34&since_id=<本地收到的最大的message_id>`
2. 这个接口利用 GORM 走联合主键索引扫描，极速拉取自上次断网这段时间以来错失的所有增量消息（`id > since_id`）。
3. 前端拿到返回的数据将其插入本地 SQLite 并刷新界面，消息便能 100% 完整接合，从而实现了微信级别的**一致性和最终可靠交付**。

---

## 5. 广场匹配结束后的控制指令广播

在广场模块中，一旦房主点击了 `匹配确认 (match-confirm)`，其他正在等待和房主聊天的用户连接会被中断。服务端会向他们发送一条特殊控制帧：

```json
{
  "type": "match_ended",
  "content": "对方已在广场找到其他伙伴，聊天室解散~"
}
```
收到此帧时，前端请直接关闭当前与该老人的长连接临时会话框，并弹出系统提示即可。
