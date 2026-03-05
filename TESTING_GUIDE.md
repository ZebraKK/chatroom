# 终端聊天室测试指南

## 当前状态

✅ **服务端运行中**：端口 8443
✅ **客户端已编译**：支持登录/注册
✅ **P0问题已修复**：
  - X25519 + Ed25519 双密钥方案
  - 签名验证
  - 服务端时间戳
  - 消息包装格式

## 已修复的问题

### 1. 消息格式问题 ✅
- **问题**：服务端响应未包装成 `Message` 格式
- **修复**：所有handler统一使用 `{type: "xxx", data: {...}}` 格式

### 2. 注册/登录问题 ✅
- **问题**：每次启动创建新用户（xw_1, xw_2...）
- **修复**：公钥匹配时视为登录，复用现有用户

### 3. 公钥获取问题 ✅
- **问题**：客户端无法获取在线用户公钥，无法发送消息
- **修复**：注册成功后自动请求在线用户公钥

## 测试步骤

### 准备工作

1. **确认服务端运行**
   ```bash
   ps aux | grep chatroom-server
   # 应该看到一个进程
   ```

2. **查看服务端日志**
   ```bash
   cd /Users/xiaowyu/xwill/chatroom/chatroom-server
   tail -f server.log
   ```

### 测试1：单用户登录

**终端1：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client
```

**预期输出：**
```
📂 Loading configuration from ~/.chatroom/config.json
✅ Keys loaded successfully
🔌 Connecting to wss://localhost:8443/ws...
✅ Connected to server
📝 Registering with server...
✅ User logged in: xw
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Terminal Chatroom Client v1.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: xw
Online users: (none)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Type /help for commands
>
```

**测试输入：**
```
Hello
```

**预期输出：**
```
❌ No other users online
```

这是正常的，因为只有一个用户在线。

### 测试2：双用户聊天 🎯

**终端1（保持运行）：** xw

**终端2（新打开）：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice
```

**首次运行预期输出（终端2）：**
```
🎉 Welcome! This is your first time running the chatroom client.

🔑 Generating cryptographic key pairs...
✅ Keys saved to:
   - /Users/xiaowyu/.chatroom/keys/alice_signing.key
   - /Users/xiaowyu/.chatroom/keys/alice_encrypt.key

✅ Keys loaded successfully
🔌 Connecting to wss://localhost:8443/ws...
✅ Connected to server
📝 Registering with server...
🆕 User registered: alice
📥 Requesting public keys for 1 online users...
✅ Received public keys for online users
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Terminal Chatroom Client v1.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: alice
Online users: xw
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Type /help for commands
>
```

**同时，终端1应该显示：**
```
👤 alice is now online
```

**在终端2（alice）输入：**
```
Hi xw! This is Alice.
```

**终端1（xw）应该收到：**
```
[alice] Hi xw! This is Alice.
```

**在终端1（xw）输入：**
```
Hello Alice! Nice to meet you!
```

**终端2（alice）应该收到：**
```
[xw] Hello Alice! Nice to meet you!
```

### 测试3：验证加密

**服务端日志应该显示：**
```
📨 Message forwarded: from=alice, recipients=1
✅ Signature verified: from=alice
📅 Server timestamp added: 1772633850 (client: 1772633850)
```

**注意**：服务端日志中**不会显示消息内容**，证明端到端加密生效！

### 测试4：命令测试

**在任一客户端输入：**
```
/help
```

**预期输出：**
```
Available commands:
  /help          - Show this help message
  /users         - List online users
  /history [n]   - Show last n messages (default: 20)
  /quit          - Exit the chatroom
```

**输入：**
```
/users
```

**预期输出（以alice为例）：**
```
Online users (1):
  - xw
```

## 验证清单

### P0功能验证
- [ ] ✅ 双密钥生成（Ed25519 + X25519）
- [ ] ✅ 签名验证（服务端拒绝伪造签名）
- [ ] ✅ 服务端时间戳（消息按服务端时间排序）
- [ ] ✅ 消息加密传输（服务端无法解密）

### P1功能验证
- [ ] ✅ 历史消息查询（`/history`）
- [ ] ⏳ 优雅关闭（Ctrl+C后数据保存）
- [ ] ⏳ 连接数限制（101个连接被拒绝）
- [ ] ⏳ 消息大小限制（65KB消息被拒绝）
- [ ] ⏳ 速率限制（10条/秒）

### 核心功能验证
- [ ] ✅ 用户注册（首次运行）
- [ ] ✅ 用户登录（再次运行，公钥匹配）
- [ ] ✅ 发送消息
- [ ] ✅ 接收消息
- [ ] ✅ 在线用户列表
- [ ] ✅ 用户上线通知
- [ ] ✅ 用户下线通知

## 故障排查

### 问题1：无法连接服务器
```bash
# 检查服务端是否运行
ps aux | grep chatroom-server

# 检查端口
lsof -i :8443

# 重启服务端
cd /Users/xiaowyu/xwill/chatroom/chatroom-server
pkill chatroom-server
nohup ./bin/chatroom-server > server.log 2>&1 &
```

### 问题2：消息发送失败 "No other users online"
- 确保至少有两个客户端同时在线
- 检查客户端是否收到公钥：日志中应有 "✅ Received public keys"

### 问题3：收不到消息
- 检查服务端日志是否有 "📨 Message forwarded"
- 检查客户端是否有签名验证失败的警告
- 确保两个客户端使用不同的用户名

## 查看日志

**服务端日志：**
```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

**关键日志标识：**
- `✅ User logged in` - 登录成功
- `🆕 User registered` - 新用户注册
- `📨 Message forwarded` - 消息转发
- `✅ Signature verified` - 签名验证通过
- `❌ Signature verification failed` - 签名验证失败

## 下一步测试

完成基础测试后，可以测试：
1. **历史消息** - `/history 10`
2. **多人聊天** - 同时运行3个客户端
3. **重连** - 断开后重新连接
4. **大消息** - 发送长文本
5. **特殊字符** - 测试中文、emoji等

## 当前已知限制

1. ⚠️ **单终端运行** - 暂不支持多终端UI（消息显示会被输入打断）
2. ⚠️ **无消息历史** - 重启客户端后不显示历史消息（需手动 `/history`）
3. ⚠️ **无离线消息** - 离线时的消息不会被保存

这些将在v2.0中改进。
