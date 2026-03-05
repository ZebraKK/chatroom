# 修复：用户下线通知错误

## 问题描述

**症状**：当一个客户端关闭后，其他在线客户端显示错误：
```
❌ Failed to unmarshal user offline notification: unexpected end of JSON input
```

**根本原因**：服务端发送用户下线通知时，**没有包装成 `Message` 格式**

## 问题定位

### 错误的代码（server.go 第128-136行）

```go
// ❌ 错误：直接序列化内层数据
notification := protocol.UserOfflineNotification{
    Type:     "user_offline",
    Username: client.Username,
}
data, _ := json.Marshal(notification)
s.connManager.Broadcast(data)
```

**发送的JSON：**
```json
{
  "type": "user_offline",
  "username": "alice"
}
```

### 客户端期望的格式

客户端的消息处理器期望所有消息都是 `Message` 格式：
```json
{
  "type": "user_offline",
  "data": {
    "type": "user_offline",
    "username": "alice"
  }
}
```

### 为什么报错？

客户端代码：
```go
func (h *Handler) handleUserOffline(msg *protocol.Message) {
    var notification protocol.UserOfflineNotification
    if err := json.Unmarshal(msg.Data, &notification); err != nil {  // ← 这里出错
        log.Printf("❌ Failed to unmarshal user offline notification: %v", err)
        return
    }
    // ...
}
```

由于服务端发送的格式错误，`msg.Data` 是空的（或者格式不对），导致 `Unmarshal` 失败。

## 修复方案

### 正确的代码

```go
// ✅ 正确：包装成 Message 格式
notification := protocol.UserOfflineNotification{
    Type:     "user_offline",
    Username: client.Username,
}
notifyData, _ := json.Marshal(notification)
envelope := protocol.Message{
    Type: "user_offline",
    Data: json.RawMessage(notifyData),
}
data, _ := json.Marshal(envelope)
s.connManager.Broadcast(data)
```

**发送的JSON：**
```json
{
  "type": "user_offline",
  "data": {
    "type": "user_offline",
    "username": "alice"
  }
}
```

## 修改的文件

- `chatroom-server/internal/server/server.go` (第128-141行)
  - 将用户下线通知包装成 `Message` 格式

## 相关修复

这是同一类问题的最后一个修复。之前已经修复了：

1. ✅ RegisterResponse 包装（register.go）
2. ✅ UserOnlineNotification 包装（register.go）
3. ✅ PubKeyResponse 包装（pubkey.go）
4. ✅ HistoryResponse 包装（history.go）
5. ✅ ChatMessage 转发包装（message.go）
6. ✅ **UserOfflineNotification 包装**（server.go）← 本次修复

**现在所有服务端响应都统一使用 `{type, data}` 格式！**

## 测试验证

### 准备工作

1. **重新编译服务端**
   ```bash
   cd /Users/xiaowyu/xwill/chatroom/chatroom-server
   go build -o bin/chatroom-server ./cmd/server
   ```

2. **重启服务端**
   ```bash
   pkill chatroom-server
   ./bin/chatroom-server
   ```

### 测试步骤

#### 步骤1：启动三个客户端

**终端1（xw）：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client
```

**终端2（alice）：**
```bash
./bin/chatroom-client -username alice
```

**终端3（bob）：**
```bash
./bin/chatroom-client -username bob
```

**预期状态：**
- 每个终端都显示其他两个用户在线
- 没有任何错误消息

#### 步骤2：关闭一个客户端

**在终端2（alice）按 `Ctrl+C` 或输入 `/quit`**

**预期结果：**

**终端1（xw）显示：**
```
👋 alice left the chat
>
```

**终端3（bob）显示：**
```
👋 alice left the chat
>
```

**✅ 重点验证：没有出现错误消息**

之前会出现：
```
❌ Failed to unmarshal user offline notification: unexpected end of JSON input
```

现在应该**没有任何错误**，只显示用户离开的通知。

#### 步骤3：验证剩余用户可以继续聊天

**在终端3（bob）输入：**
```
Alice just left!
```

**终端1（xw）应该收到：**
```
[23:55:30] bob: Alice just left!
>
```

**在终端1（xw）输入：**
```
Yes, I saw that.
```

**终端3（bob）应该收到：**
```
[23:55:35] xw: Yes, I saw that.
>
```

✅ **如果消息正常收发，说明修复成功！**

### 服务端日志验证

```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

**应该看到：**
```
✅ User logged in: xw
✅ User logged in: alice
✅ User logged in: bob
👋 Client disconnected: id=..., username=alice
📨 Message forwarded: from=bob, recipients=1
📨 Message forwarded: from=xw, recipients=1
```

**不应该看到任何错误消息。**

## 验证清单

### ✅ 功能验证

- [ ] 三个客户端成功登录
- [ ] 一个客户端退出时，其他客户端显示 "left the chat"
- [ ] **没有出现 "Failed to unmarshal" 错误**
- [ ] 剩余客户端可以继续聊天
- [ ] 再次启动退出的客户端，其他客户端显示 "joined the chat"

### ✅ 消息格式验证

所有服务端响应现在都使用统一格式：
```json
{
  "type": "message_type",
  "data": { ... }
}
```

包括：
- `register_response`
- `user_online`
- `user_offline` ← 本次修复
- `message`
- `pubkeys`
- `history_response`

## 总结

**问题**：用户下线通知未包装成 `Message` 格式
**影响**：客户端无法正确解析，显示错误消息
**修复**：将 `UserOfflineNotification` 包装成 `{type, data}` 格式
**结果**：用户下线通知正常显示，无错误

---

**更新时间**：2026-03-04 23:52
**状态**：✅ 已修复
**服务端版本**：`chatroom-server/bin/chatroom-server` (2026-03-04 23:52)
