# 三客户端终端布局建议

## 推荐布局（分屏）

```
┌─────────────────────────────┬─────────────────────────────┐
│                             │                             │
│   终端1: xw                  │   终端2: alice              │
│   (默认用户)                 │   (第二用户)                │
│                             │                             │
│   > ./bin/chatroom-client   │   > ./bin/chatroom-client   │
│                             │      -username alice        │
│                             │                             │
│   Logged in as: xw          │   Logged in as: alice       │
│   Online users: alice, bob  │   Online users: xw, bob     │
│                             │                             │
│   >                         │   >                         │
│                             │                             │
├─────────────────────────────┴─────────────────────────────┤
│                                                           │
│   终端3: bob                                              │
│   (第三用户)                                              │
│                                                           │
│   > ./bin/chatroom-client -username bob                  │
│                                                           │
│   Logged in as: bob                                       │
│   Online users: xw, alice                                 │
│                                                           │
│   >                                                       │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

## 快速启动步骤

### 方法1：使用快速启动脚本

**终端1：**
```bash
cd /Users/xiaowyu/xwill/chatroom
./QUICK_START.sh xw
```

**终端2（新窗口）：**
```bash
cd /Users/xiaowyu/xwill/chatroom
./QUICK_START.sh alice
```

**终端3（新窗口）：**
```bash
cd /Users/xiaowyu/xwill/chatroom
./QUICK_START.sh bob
```

---

### 方法2：手动启动

**终端1：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client
```

**终端2：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice
```

**终端3：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username bob
```

---

## macOS Terminal 分屏方法

### 使用 iTerm2（推荐）

1. 打开 iTerm2
2. **第一个窗口**：
   - 运行 `./bin/chatroom-client`
3. **分割屏幕**：
   - `Cmd + D` - 垂直分割
   - 运行 `./bin/chatroom-client -username alice`
4. **再次分割**：
   - 选中第一个窗格
   - `Cmd + Shift + D` - 水平分割
   - 运行 `./bin/chatroom-client -username bob`

### 使用 tmux

```bash
# 创建新会话
tmux new -s chatroom

# 水平分割
Ctrl+b "

# 垂直分割
Ctrl+b %

# 切换窗格
Ctrl+b ←→↑↓

# 在每个窗格运行客户端
# 窗格1: ./bin/chatroom-client
# 窗格2: ./bin/chatroom-client -username alice
# 窗格3: ./bin/chatroom-client -username bob
```

### 使用系统 Terminal

1. 打开三个独立的 Terminal 窗口
2. 使用 **Mission Control** 或 **Spaces** 排列窗口
3. 在每个窗口分别运行客户端

---

## 完整演示对话脚本

### 初始状态

```
[xw]                        [alice]                     [bob]
Logged in as: xw            Logged in as: alice         Logged in as: bob
Online users: (none)        Online users: xw            Online users: xw, alice
>                           >                           >
```

---

### 对话序列1：Bob 问候

**Bob 输入：** `Hi everyone! I just joined.`

```
[xw]                        [alice]                     [bob]
[bob] Hi everyone!          [bob] Hi everyone!          > Hi everyone! I just joined.
I just joined.              I just joined.
>                           >                           >
```

---

### 对话序列2：Alice 回应

**Alice 输入：** `Welcome Bob! 👋`

```
[xw]                        [alice]                     [bob]
[bob] Hi everyone!          [bob] Hi everyone!          [alice] Welcome Bob! 👋
I just joined.              I just joined.
[alice] Welcome Bob! 👋     > Welcome Bob! 👋           >
>                           >
```

---

### 对话序列3：Xw 发起讨论

**Xw 输入：** `Great! Let's test the end-to-end encryption.`

```
[xw]                        [alice]                     [bob]
[alice] Welcome Bob! 👋     [alice] Welcome Bob! 👋     [alice] Welcome Bob! 👋
> Great! Let's test the     [xw] Great! Let's test      [xw] Great! Let's test
end-to-end encryption.      the end-to-end encryption.  the end-to-end encryption.
>                           >                           >
```

---

### 对话序列4：查看在线用户

**所有人输入：** `/users`

```
[xw]                        [alice]                     [bob]
> /users                    > /users                    > /users
Online users (2):           Online users (2):           Online users (2):
- alice                     - xw                        - xw
- bob                       - bob                       - alice
>                           >                           >
```

---

### 对话序列5：Bob 退出

**Bob 输入：** `/quit`

```
[xw]                        [alice]                     [bob]
👋 bob is now offline       👋 bob is now offline       > /quit
>                           >                           👋 Goodbye!
                                                        [已退出]
```

---

## 验证要点

### ✅ 功能验证

1. **消息广播**：一人发送，其他人都能收到
2. **用户通知**：上线/下线都有通知
3. **命令功能**：`/users`, `/help` 正常工作
4. **加密验证**：服务端日志看不到消息内容

### ✅ 服务端日志

```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

应该看到：
```
✅ User logged in: xw
🆕 User registered: alice
🆕 User registered: bob
📨 Message forwarded: from=bob, recipients=2
✅ Signature verified: from=bob
📨 Message forwarded: from=alice, recipients=2
✅ Signature verified: from=alice
📨 Message forwarded: from=xw, recipients=2
✅ Signature verified: from=xw
👋 Client disconnected: username=bob
```

**注意**：消息内容**不会出现**在日志中！

---

## 高级测试

### 测试1：历史消息

**在任一客户端输入：**
```
/history 10
```

应该显示最近10条消息。

### 测试2：断线重连

1. 关闭 alice（Ctrl+C）
2. 其他用户看到 `👋 alice is now offline`
3. 重新启动 alice
4. 其他用户看到 `👤 alice is now online`
5. Alice 看到之前的在线用户

### 测试3：性能测试

连续发送多条消息，测试速率限制（10条/秒）。

---

## 常见问题

### Q: 如何切换到其他用户？
**A:** 退出当前客户端（`/quit` 或 `Ctrl+C`），使用不同的 `-username` 参数重新启动。

### Q: 能否同时以相同用户名登录？
**A:** 可以，但只有**公钥相同**的才能复用用户名。不同的密钥会分配新用户名（如 alice_1）。

### Q: 密钥文件在哪里？
**A:** `~/.chatroom/keys/`
- `{username}_signing.key` - Ed25519 签名私钥
- `{username}_encrypt.key` - X25519 加密私钥

### Q: 如何清空聊天记录？
**A:** 删除服务端数据文件：
```bash
rm /Users/xiaowyu/xwill/chatroom/chatroom-server/data/messages.jsonl
rm /Users/xiaowyu/xwill/chatroom/chatroom-server/data/users.json
```
然后重启服务端。

---

**准备好了吗？现在就开始三客户端演示吧！** 🚀
