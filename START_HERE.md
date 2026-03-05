# 🚀 从这里开始 - 三客户端演示

## 一键启动（最简单）

### 第1步：确认服务端运行

```bash
ps aux | grep chatroom-server
```

如果没有看到进程，启动服务端：
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-server
./bin/chatroom-server &
```

---

### 第2步：开启三个终端窗口

**终端1 - 用户 xw：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client
```

**终端2 - 用户 alice（新窗口）：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice
```

**终端3 - 用户 bob（新窗口）：**
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username bob
```

---

### 第3步：开始对话

**在 bob 的终端输入：**
```
Hello everyone!
```

**查看其他两个终端**，应该都收到：
```
[bob] Hello everyone!
```

**在 alice 的终端输入：**
```
Hi Bob! Welcome to the chatroom.
```

**查看 xw 和 bob 的终端**，应该都收到：
```
[alice] Hi Bob! Welcome to the chatroom.
```

**在 xw 的终端输入：**
```
Nice to meet you all!
```

**查看 alice 和 bob 的终端**，应该都收到：
```
[xw] Nice to meet you all!
```

---

## 完成！🎉

现在您已经成功运行了三客户端聊天演示！

### 可以尝试的命令：

```bash
/users      # 查看在线用户
/history    # 查看历史消息
/help       # 查看帮助
/quit       # 退出
```

---

## 预期看到的效果

### 终端1（xw）
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: xw
Online users: alice, bob
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[bob] Hello everyone!
[alice] Hi Bob! Welcome to the chatroom.
> Nice to meet you all!
>
```

### 终端2（alice）
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: alice
Online users: xw, bob
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[bob] Hello everyone!
> Hi Bob! Welcome to the chatroom.
[xw] Nice to meet you all!
>
```

### 终端3（bob）
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Logged in as: bob
Online users: xw, alice
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
> Hello everyone!
[alice] Hi Bob! Welcome to the chatroom.
[xw] Nice to meet you all!
>
```

---

## 查看服务端日志（可选）

**新开一个终端：**
```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

**应该看到：**
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
```

**注意**：日志中看不到消息内容，因为是**端到端加密**！🔒

---

## 故障排查

### 问题1：显示 "No other users online"
**原因**：只有一个客户端在线

**解决**：至少需要启动2个客户端

---

### 问题2：连接失败
**原因**：服务端未运行

**解决**：
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-server
./bin/chatroom-server
```

---

### 问题3：alice/bob 首次启动没有生成密钥
**原因**：配置文件已存在

**解决**：使用 `-username` 参数可以为不同用户生成不同的密钥

---

## 更多信息

- 详细测试指南：[DEMO_3_CLIENTS.md](./DEMO_3_CLIENTS.md)
- 终端布局建议：[TERMINAL_LAYOUT.md](./TERMINAL_LAYOUT.md)
- 完整测试指南：[TESTING_GUIDE.md](./TESTING_GUIDE.md)

---

**就是这么简单！享受您的加密聊天室吧！** 🎊
