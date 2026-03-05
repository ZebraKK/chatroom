# 测试聊天功能

## 步骤

### 1. 启动第一个客户端（终端1）
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client
```

预期：
- 登录成功显示 "✅ User logged in: xw"
- 显示 "No other users online"

### 2. 启动第二个客户端（终端2）
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
./bin/chatroom-client -username alice
```

预期：
- 第一次运行会生成新密钥
- 注册成功显示 "User registered: alice"
- 显示 "Online users: xw"
- **终端1** 应该收到 "alice is now online"

### 3. 发送消息测试

**在终端1（xw）输入：**
```
Hello Alice!
```

**在终端2（alice）应该看到：**
```
[xw] Hello Alice!
```

**在终端2（alice）输入：**
```
Hi xw, nice to meet you!
```

**在终端1（xw）应该看到：**
```
[alice] Hi xw, nice to meet you!
```

## 预期结果

✅ 双方可以互相发送和接收消息
✅ 消息经过端到端加密
✅ 服务端无法解密消息内容

## 查看服务端日志

```bash
tail -f /Users/xiaowyu/xwill/chatroom/chatroom-server/server.log
```

应该看到：
- ✅ User logged in: xw
- 🆕 User registered: alice
- 📨 Message forwarded: from=xw, recipients=1
- 📨 Message forwarded: from=alice, recipients=1
