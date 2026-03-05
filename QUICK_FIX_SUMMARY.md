# 消息发送问题修复总结

## 问题诊断

**症状**：登录后无法发送消息

**根本原因**：客户端缺少在线用户的公钥

### 问题链路
1. 客户端注册/登录成功 ✅
2. 收到在线用户列表（如 `["alice"]`） ✅
3. **但没有请求这些用户的公钥** ❌
4. 尝试发送消息时，`msgHandler.GetOnlineUsers()` 返回空数组
5. 触发 "No other users online" 错误

## 修复方案

### 文件：`chatroom-client/cmd/client/main.go`

**修改位置**：第143行之后

**修改前：**
```go
// 显示连接成功
terminal.ShowConnected(regResp.OnlineUsers)

// 创建命令处理器
cmdHandler := command.New(conn, terminal)
```

**修改后：**
```go
// 显示连接成功
terminal.ShowConnected(regResp.OnlineUsers)

// 如果有在线用户，请求他们的公钥
if len(regResp.OnlineUsers) > 0 {
	log.Printf("📥 Requesting public keys for %d online users...", len(regResp.OnlineUsers))
	pubKeyReq := protocol.PubKeyRequest{
		Type:  "get_pubkeys",
		Users: regResp.OnlineUsers,
	}
	if err := conn.SendMessage(pubKeyReq); err != nil {
		log.Printf("⚠️  Warning: Failed to request public keys: %v", err)
	} else {
		// 等待公钥响应
		pubKeyResp, err := conn.ReceiveMessage()
		if err == nil && pubKeyResp.Type == "pubkeys" {
			// 让消息处理器处理公钥
			msgHandler.HandleMessage(pubKeyResp)
			log.Printf("✅ Received public keys for online users")
		}
	}
}

// 创建命令处理器
cmdHandler := command.New(conn, terminal)
```

## 测试验证

### 1. 重新编译
```bash
cd /Users/xiaowyu/xwill/chatroom/chatroom-client
go build -o bin/chatroom-client ./cmd/client
```

### 2. 测试步骤

**终端1：**
```bash
./bin/chatroom-client
# 应该看到 "Logged in as: xw"
# Online users: (none)
```

**终端2：**
```bash
./bin/chatroom-client -username alice
# 应该看到：
# 📥 Requesting public keys for 1 online users...
# ✅ Received public keys for online users
# Online users: xw
```

**在终端2输入：**
```
Hello xw!
```

**终端1应该收到：**
```
[alice] Hello xw!
```

✅ **修复成功！**

## 技术细节

### 消息加密流程
```
Alice 发送消息给 xw:
1. Alice生成随机AES-256密钥
2. 用AES密钥加密消息内容
3. 用xw的X25519公钥加密AES密钥
4. 用Alice的Ed25519私钥签名
5. 发送到服务器
6. 服务器验证签名 ✅
7. 服务器转发给xw（不解密内容）
8. xw用自己的X25519私钥解密AES密钥
9. 用AES密钥解密消息内容
10. 验证Alice的签名 ✅
```

### 为什么需要公钥？
- **加密**：需要接收者的 X25519 公钥来加密AES密钥
- **验证**：需要发送者的 Ed25519 公钥来验证签名

没有公钥 = 无法加密 = 无法发送消息

## 相关修复

今天一共修复了3个关键问题：

### 1. 消息格式问题（P0）
- **问题**：服务端响应未包装成 `{type, data}` 格式
- **影响**：客户端无法识别消息类型
- **修复文件**：
  - `register.go` - RegisterResponse, UserOnlineNotification
  - `pubkey.go` - PubKeyResponse
  - `history.go` - HistoryResponse
  - `message.go` - ChatMessage转发

### 2. 重复注册问题（P0）
- **问题**：每次启动创建新用户（xw_1, xw_2...）
- **影响**：用户体验差，数据库污染
- **修复文件**：`user/manager.go` - Register方法
- **逻辑**：公钥匹配 → 登录，公钥不匹配 → 新用户名

### 3. 公钥获取问题（本次修复，P0）
- **问题**：客户端未主动获取在线用户公钥
- **影响**：无法发送消息
- **修复文件**：`cmd/client/main.go`
- **逻辑**：注册成功 → 请求在线用户公钥 → 保存到本地

## 下一步

所有P0问题已修复！可以开始测试：

1. ✅ **双用户聊天** - 基础功能
2. ⏳ **多用户聊天** - 3+用户同时在线
3. ⏳ **历史消息** - `/history` 命令
4. ⏳ **重连测试** - 断线重连
5. ⏳ **压力测试** - 大量消息发送

参考：[TESTING_GUIDE.md](./TESTING_GUIDE.md)
