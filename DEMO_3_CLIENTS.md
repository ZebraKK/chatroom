# 三客户端演示指南

## 前提条件

✅ 服务端运行在 `localhost:8443`
✅ 客户端已编译：`chatroom-client/bin/chatroom-client`

## 开启三个终端窗口

### 终端1：启动第一个客户端（用户 xw）

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

---

### 终端2：启动第二个客户端（用户 alice）

**新开一个终端窗口，运行：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice
```

**首次运行预期输出：**
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

**同时，终端1（xw）会显示：**
```
👤 alice is now online
```

---

### 终端3：启动第三个客户端（用户 bob）

**再开一个新终端窗口，运行：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username bob
```

**首次运行预期输出：**
```
🎉 Welcome! This is your first time running the chatroom client.

🔑 Generating cryptographic key pairs...
✅ Keys saved to:
   - /Users/xiaowyu/.chatroom/keys/bob_signing.key
   - /Users/xiaowyu/.chatroom/keys/bob_encrypt.key

✅ Keys loaded successfully
🔌 Connecting to wss://localhost:8443/ws...
✅ Connected to server
📝 Registering with server...
🆕 User registered: bob
📥 Requesting public keys for 2 online users...
✅ Received public keys for online users
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Terminal Chatroom Client v1.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: bob
Online users: xw, alice
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Type /help for commands
>
```

**同时：**
- **终端1（xw）会显示：** `👤 bob is now online`
- **终端2（alice）会显示：** `👤 bob is now online`

---

## 演示对话场景

### 场景1：Bob 向所有人问好

**在终端3（bob）输入：**
```
Hello everyone! I'm Bob.
```

**终端1（xw）收到：**
```
[bob] Hello everyone! I'm Bob.
```

**终端2（alice）收到：**
```
[bob] Hello everyone! I'm Bob.
```

---

### 场景2：Alice 回复

**在终端2（alice）输入：**
```
Hi Bob! Welcome to the chatroom.
```

**终端1（xw）收到：**
```
[alice] Hi Bob! Welcome to the chatroom.
```

**终端3（bob）收到：**
```
[alice] Hi Bob! Welcome to the chatroom.
```

---

### 场景3：Xw 发起讨论

**在终端1（xw）输入：**
```
Great! Now we have 3 people online. Let's test the encryption!
```

**终端2（alice）收到：**
```
[xw] Great! Now we have 3 people online. Let's test the encryption!
```

**终端3（bob）收到：**
```
[xw] Great! Now we have 3 people online. Let's test the encryption!
```

---

### 场景4：测试命令

**在任一终端输入：**
```
/users
```

**预期输出：**
```
Online users (2):
  - xw
  - alice
  - bob
```
（显示除自己外的其他用户）

---

### 场景5：Alice 退出

**在终端2（alice）输入：**
```
/quit
```

或者直接按 `Ctrl+C`

**终端1（xw）和终端3（bob）会显示：**
```
👋 alice is now offline
```

---

## 关键点说明

### 1. 密钥文件位置
每个用户的密钥独立存储在：
```
~/.chatroom/keys/
├── xw_signing.key
├── xw_encrypt.key
├── alice_signing.key
├── alice_encrypt.key
├── bob_signing.key
└── bob_encrypt.key
```

### 2. 配置文件
默认配置文件：`~/.chatroom/config.json`

**如果需要完全独立的配置**，可以手动指定：
```bash
# 方法1：使用环境变量（需要修改代码支持）
HOME=/tmp/alice ./bin/chatroom-client -username alice

# 方法2：修改用户名参数即可（推荐）
./bin/chatroom-client -username alice
```

### 3. 再次登录（第二次运行）

**终端2再次运行：**
```bash
./bin/chatroom-client -username alice
```

**预期输出：**
```
📂 Loading configuration from ~/.chatroom/config.json
✅ Keys loaded successfully
🔌 Connecting to wss://localhost:8443/ws...
✅ Connected to server
📝 Registering with server...
✅ User logged in: alice  ← 注意这里是"登录"不是"注册"
📥 Requesting public keys for 2 online users...
✅ Received public keys for online users
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: alice
Online users: xw, bob
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
>
```

---

## 服务端日志验证

**在另一个终端查看服务端日志：**
```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

**应该看到：**
```
2026/03/04 22:45:01 ✅ Client connected: id=abc123...
2026/03/04 22:45:01 ✅ User logged in: xw
2026/03/04 22:45:10 ✅ Client connected: id=def456...
2026/03/04 22:45:10 🆕 User registered: alice
2026/03/04 22:45:20 ✅ Client connected: id=ghi789...
2026/03/04 22:45:20 🆕 User registered: bob
2026/03/04 22:45:30 📨 Message forwarded: from=bob, recipients=2
2026/03/04 22:45:30 ✅ Signature verified: from=bob
2026/03/04 22:45:40 📨 Message forwarded: from=alice, recipients=2
2026/03/04 22:45:40 ✅ Signature verified: from=alice
2026/03/04 22:45:50 📨 Message forwarded: from=xw, recipients=2
2026/03/04 22:45:50 ✅ Signature verified: from=xw
```

**注意**：日志中**不会显示消息内容**，证明端到端加密生效！

---

## 完整演示对话示例

```
[终端1 - xw]                [终端2 - alice]             [终端3 - bob]
─────────────────────────────────────────────────────────────────────
Logged in as: xw
Online users: (none)
>
                            👤 alice is now online
                            Logged in as: alice
                            Online users: xw
                            >
                                                        👤 bob is now online
👤 bob is now online        👤 bob is now online        Logged in as: bob
                                                        Online users: xw, alice
                                                        > Hello everyone! I'm Bob.
[bob] Hello everyone!       [bob] Hello everyone!
I'm Bob.                    I'm Bob.
                            > Hi Bob! Welcome!
[alice] Hi Bob! Welcome!                                [alice] Hi Bob! Welcome!
> Great! Let's chat!
                            [xw] Great! Let's chat!     [xw] Great! Let's chat!
                            > /users
                            Online users (2):
                            - xw
                            - bob
                            > /quit
👋 alice is now offline                                 👋 alice is now offline
```

---

## 快速启动命令汇总

```bash
# 准备工作：确保服务端运行
cd /Users/xiaowyu/xwill/chatroom/chatroom-server
ps aux | grep chatroom-server  # 检查服务端是否运行

# 终端1
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client

# 终端2（新窗口）
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice

# 终端3（新窗口）
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username bob
```

---

## 故障排查

### 问题：提示 "username taken"
**原因**：用户名已被其他客户端使用（公钥不匹配）

**解决**：使用不同的用户名
```bash
./bin/chatroom-client -username carol
```

### 问题：无法发送消息 "No other users online"
**原因**：只有一个客户端在线

**解决**：至少需要2个客户端在线才能互相发送消息

### 问题：收不到消息
**检查**：
1. 确认服务端日志有 "Message forwarded"
2. 确认接收方客户端在线
3. 确认日志有 "Received public keys"

---

现在您可以开始三客户端演示了！🚀
