# 终端聊天室实施计划

## 文档信息
- **创建时间**: 2026-03-02
- **项目**: 终端聊天室 (Terminal Chatroom)
- **版本**: v1.0
- **状态**: 🚧 开发中

---

## 1. 执行摘要

本文档提供终端聊天室项目的详细实施计划，基于架构评估发现的关键问题制定。

### 架构质量评分: 81/100

| 维度 | 评分 | 说明 |
|------|------|------|
| 模块完整性 | 85/100 | 核心模块齐全，但缺少历史消息、签名验证等 |
| 加密方案 | 70/100 | 设计思路正确，但Ed25519加密技术上不可行 |
| 并发模型 | 80/100 | 设计清晰合理，但缺少优雅关闭和资源限制 |
| 架构一致性 | 90/100 | 客户端和服务端高度一致，少量不一致点 |
| 安全性 | 75/100 | E2EE设计良好，但签名验证缺失、时间戳不可信 |

---

## 2. 关键问题分类

### P0级问题（阻塞性，必须修复）

#### 问题1: Ed25519加密方案技术错误 🔴
- **位置**: CLIENT_ARCHITECTURE.md (第253行 `encryptAESKey`)
- **问题**: Ed25519是签名算法，不能直接用于加密AES密钥
- **影响**: 核心加密功能无法实现
- **解决方案**: 采用X25519 + Ed25519双密钥方案

#### 问题2: 服务端签名验证缺失 🔴
- **位置**: SERVER_ARCHITECTURE.md (第616行 `// TODO: 验证签名`)
- **问题**: MessageHandler未验证消息签名
- **影响**: 严重安全漏洞，可伪造消息发送者
- **解决方案**: 实现完整的Ed25519签名验证流程

#### 问题3: 消息时间戳权威性问题 🔴
- **问题**: 使用客户端生成的时间戳，可被篡改
- **影响**: 消息顺序可被伪造，影响历史记录可信度
- **解决方案**: 双时间戳方案（client_timestamp + server_timestamp）

### P1级问题（功能完整性）

#### 问题4: 历史消息查询功能缺失 🟡
- **需求**: 需求文档要求 `/history [n]` 命令
- **现状**: 架构中完全没有设计
- **解决方案**: 新增HistoryHandler和相关协议

#### 问题5: 优雅关闭机制未实现 🟡
- **位置**: SERVER_ARCHITECTURE.md 第901-907行只有注释
- **解决方案**: 完整实现信号处理、连接关闭、数据保存

#### 问题6: 基础安全限制缺失 🟡
- **问题**: 缺少连接数限制、消息大小限制
- **解决方案**: 实现ConnectionLimiter和速率限制

---

## 3. 技术方案详解

### 方案1: X25519 + Ed25519 双密钥方案（P0）

#### 技术原理
- **Ed25519**: 用于签名验证（保持不变）
- **X25519**: (Curve25519 ECDH) 用于加密AES密钥
- 两者都基于Curve25519，密钥长度一致（32字节）

#### 密钥结构修改

```go
// 修改前（错误）
type KeyPair struct {
    PrivateKey ed25519.PrivateKey  // 仅用于签名
    PublicKey  ed25519.PublicKey
}

// 修改后（正确）
type KeyPair struct {
    // 签名密钥对
    SigningPrivate ed25519.PrivateKey
    SigningPublic  ed25519.PublicKey

    // 加密密钥对（X25519）
    EncryptPrivate [32]byte
    EncryptPublic  [32]byte
}
```

#### 协议定义修改

```json
// 注册请求（修改后）
{
  "type": "register",
  "username": "alice",
  "signing_key": "base64_ed25519_public",
  "encryption_key": "base64_x25519_public",
  "algorithm": "ed25519+x25519"
}

// 公钥响应（修改后）
{
  "type": "pubkeys",
  "keys": {
    "bob": {
      "signing_key": "base64...",
      "encryption_key": "base64...",
      "algorithm": "ed25519+x25519"
    }
  }
}
```

#### AES密钥加密实现

```go
func encryptAESKey(aesKey []byte, recipientX25519Pub [32]byte, myX25519Private [32]byte) ([]byte, error) {
    // 1. ECDH密钥交换
    sharedSecret, _ := curve25519.X25519(myX25519Private, recipientX25519Pub)

    // 2. 使用HKDF派生加密密钥
    kdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("aes-key-wrap"))
    wrapKey := make([]byte, 32)
    kdf.Read(wrapKey)

    // 3. 使用ChaCha20-Poly1305加密AES密钥
    cipher, _ := chacha20poly1305.New(wrapKey)
    nonce := make([]byte, cipher.NonceSize())
    rand.Read(nonce)

    return cipher.Seal(nonce, nonce, aesKey, nil), nil
}
```

**依赖库**:
- `golang.org/x/crypto/curve25519`
- `golang.org/x/crypto/chacha20poly1305`
- `golang.org/x/crypto/hkdf`

---

### 方案2: 签名验证实现（P0）

#### MessageHandler实现

```go
func (h *MessageHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    var chatMsg protocol.ChatMessage
    json.Unmarshal(msg.Data, &chatMsg)

    // 1. 获取发送者的签名公钥
    signingKey, _, err := h.router.userManager.GetPublicKeys(chatMsg.From)
    if err != nil {
        return errors.New("sender not found")
    }

    // 2. 构造待签名数据（必须与客户端一致）
    signData := fmt.Sprintf("%s:%d:%s",
        chatMsg.From,
        chatMsg.ClientTimestamp,
        chatMsg.AESEncryptedMessage,
    )

    // 3. 验证Ed25519签名
    signature, _ := base64.StdEncoding.DecodeString(chatMsg.Signature)
    if !ed25519.Verify(signingKey, []byte(signData), signature) {
        log.Printf("签名验证失败: from=%s", chatMsg.From)
        return errors.New("invalid signature")
    }

    // 4. 防重放攻击：验证时间戳在合理范围内
    serverTime := time.Now().Unix()
    if abs(chatMsg.ClientTimestamp - serverTime) > 300 { // 5分钟窗口
        return errors.New("timestamp out of range")
    }

    // 继续处理消息...
}
```

#### UserManager新增方法

```go
func (m *UserManager) GetPublicKeys(username string) (signingKey ed25519.PublicKey, encryptKey [32]byte, err error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    user, exists := m.users[username]
    if !exists {
        return nil, [32]byte{}, errors.New("user not found")
    }

    return user.SigningKey, user.EncryptionKey, nil
}
```

---

### 方案3: 服务端时间戳权威性（P0）

#### ChatMessage结构体修改

```go
type ChatMessage struct {
    From                string      `json:"from"`
    ClientTimestamp     int64       `json:"client_timestamp"`   // 客户端时间（仅用于签名验证）
    ServerTimestamp     int64       `json:"server_timestamp"`   // 服务端时间（权威）
    AESEncryptedMessage string      `json:"aes_encrypted_message"`
    Recipients          []Recipient `json:"recipients,omitempty"`
    EncryptedAESKey     string      `json:"encrypted_aes_key,omitempty"`
    Signature           string      `json:"signature"`
}
```

#### MessageHandler修改

```go
func (h *MessageHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    // 解析并验证签名（使用client_timestamp）
    // ...

    // 添加服务端权威时间戳
    chatMsg.ServerTimestamp = time.Now().Unix()

    // 保存消息（使用server_timestamp排序）
    h.router.storage.SaveMessage(&chatMsg)

    // 转发给接收者（包含双时间戳）
    for _, recipient := range chatMsg.Recipients {
        individualMsg := protocol.ChatMessage{
            From:                chatMsg.From,
            ClientTimestamp:     chatMsg.ClientTimestamp,
            ServerTimestamp:     chatMsg.ServerTimestamp,  // 权威时间
            AESEncryptedMessage: chatMsg.AESEncryptedMessage,
            EncryptedAESKey:     recipient.EncryptedAESKey,
            Signature:           chatMsg.Signature,
        }
        // 发送...
    }
}
```

---

### 方案4: 历史消息查询功能（P1）

#### HistoryHandler实现

```go
// internal/handler/history.go
type HistoryHandler struct {
    router *message.Router
}

func (h *HistoryHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    if client.Username == "" {
        return errors.New("not authenticated")
    }

    var req protocol.HistoryRequest
    json.Unmarshal(msg.Data, &req)

    // 限制查询数量（默认20，最大100）
    if req.Limit <= 0 || req.Limit > 100 {
        req.Limit = 20
    }

    // 从存储加载消息
    allMessages, _ := h.router.storage.LoadMessages(0)

    // 反向遍历+时间过滤+分页
    var filtered []protocol.ChatMessage
    for i := len(allMessages) - 1; i >= 0; i-- {
        msg := allMessages[i]

        if req.Before > 0 && msg.ServerTimestamp >= req.Before {
            continue
        }

        filtered = append(filtered, *msg)
        if len(filtered) >= req.Limit {
            break
        }
    }

    // 响应
    resp := protocol.HistoryResponse{
        Type:     "history_response",
        Messages: filtered,
        HasMore:  len(filtered) == req.Limit,
    }

    data, _ := json.Marshal(resp)
    client.SendChan <- data
    return nil
}
```

#### 协议定义

```go
type HistoryRequest struct {
    Type   string `json:"type"`  // "history"
    Limit  int    `json:"limit"`
    Before int64  `json:"before,omitempty"` // 分页用
}

type HistoryResponse struct {
    Type     string        `json:"type"` // "history_response"
    Messages []ChatMessage `json:"messages"`
    HasMore  bool          `json:"has_more"`
}
```

---

### 方案5: 优雅关闭机制（P1）

```go
type Server struct {
    // 现有字段...
    shutdown chan struct{}
    wg       sync.WaitGroup
}

func (s *Server) Start() error {
    s.shutdown = make(chan struct{})

    // 注册系统信号处理
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        s.Shutdown(ctx)
        os.Exit(0)
    }()

    return http.ListenAndServeTLS(...)
}

func (s *Server) Shutdown(ctx context.Context) error {
    log.Println("开始优雅关闭...")

    // 1. 停止接受新连接
    close(s.shutdown)

    // 2. 广播服务器关闭通知
    notification := map[string]string{
        "type":    "server_shutdown",
        "message": "服务器即将关闭",
    }
    data, _ := json.Marshal(notification)
    s.connManager.Broadcast(data)

    // 3. 等待消息发送完成
    time.Sleep(1 * time.Second)

    // 4. 关闭所有WebSocket连接
    s.connManager.CloseAll()

    // 5. 保存数据
    if err := s.userManager.Save(s.storage); err != nil {
        log.Printf("保存用户数据失败: %v", err)
    }

    // 6. 等待所有goroutine退出（带超时）
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("优雅关闭完成")
        return nil
    case <-ctx.Done():
        log.Println("关闭超时，强制退出")
        return ctx.Err()
    }
}
```

---

### 方案6: 基础安全限制（P1）

```go
const (
    MaxConnections  = 100
    MaxMessageSize  = 64 * 1024  // 64KB
    MaxUsernameLen  = 32
    RateLimitPerSec = 10
)

// 连接限制器
type ConnectionLimiter struct {
    mu    sync.Mutex
    count int
}

func (l *ConnectionLimiter) Acquire() error {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.count >= MaxConnections {
        return errors.New("server at capacity")
    }

    l.count++
    return nil
}

func (l *ConnectionLimiter) Release() {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.count--
}

// WebSocket处理修改
func (s *Server) handleWebSocket(ws *websocket.Conn) {
    // 1. 检查连接限制
    if err := s.connLimiter.Acquire(); err != nil {
        ws.WriteClose(1008) // Policy Violation
        return
    }
    defer s.connLimiter.Release()

    // 2. 设置消息大小限制
    ws.MaxPayloadBytes = MaxMessageSize

    client := s.connManager.AddClient(ws)
    defer s.connManager.RemoveClient(client.ID)

    // 3. 创建速率限制器
    limiter := rate.NewLimiter(rate.Limit(RateLimitPerSec), RateLimitPerSec*2)

    for {
        // 速率限制
        if !limiter.Allow() {
            time.Sleep(100 * time.Millisecond)
            continue
        }

        var msg protocol.Message
        if err := websocket.JSON.Receive(ws, &msg); err != nil {
            break
        }

        s.messageRouter.Route(client, &msg)
    }
}
```

---

## 4. 实施时间表

### 第一阶段：P0修复（预计3天）

#### Day 1-2: Ed25519加密方案修复
- [x] 修改CLIENT_ARCHITECTURE.md和SERVER_ARCHITECTURE.md
- [ ] 更新协议定义
- [ ] 实现X25519密钥加密/解密
- [ ] 单元测试验证

#### Day 3上午: 签名验证实现
- [ ] SERVER_ARCHITECTURE.md MessageHandler实现
- [ ] 添加UserManager.GetPublicKeys方法
- [ ] 测试伪造签名被拒绝

#### Day 3下午: 服务端时间戳修复
- [ ] 修改ChatMessage协议
- [ ] MessageHandler添加server_timestamp
- [ ] 验证消息排序使用服务端时间

### 第二阶段：P1功能补充（预计3天）

#### Day 4-5: 历史消息查询功能
- [ ] 创建HistoryHandler
- [ ] 定义协议并注册路由
- [ ] 客户端实现/history命令
- [ ] 测试分页加载

#### Day 6上午: 优雅关闭机制
- [ ] 实现Server.Shutdown
- [ ] 信号处理和广播通知
- [ ] 测试数据保存

#### Day 6下午: 基础安全限制
- [ ] 实现ConnectionLimiter
- [ ] 添加消息大小和速率限制
- [ ] 压力测试验证

### 第三阶段：集成验证（预计1天）

#### Day 7: 端到端测试
- [ ] 完整消息流程测试
- [ ] 安全性测试
- [ ] 性能压力测试
- [ ] 文档更新完成度检查

---

## 5. 验证清单

### P0问题验证
- [ ] X25519成功加密/解密AES密钥（单元测试）
- [ ] 伪造签名的消息被服务端拒绝
- [ ] 时间戳过期（>5分钟）的消息被拒绝
- [ ] 消息按server_timestamp正确排序
- [ ] 完整的加密消息流程：Alice发送 → Bob接收并解密

### P1功能验证
- [ ] `/history` 命令返回最近20条消息
- [ ] `/history 50` 返回50条
- [ ] 服务器Ctrl+C后数据文件正确保存
- [ ] 客户端收到server_shutdown通知
- [ ] 第101个并发连接被拒绝
- [ ] 65KB消息被拒绝
- [ ] 速率限制生效（1秒10条）

### 安全性验证
- [ ] 无法发送未签名的消息
- [ ] 无法伪造他人消息
- [ ] 无法篡改消息时间戳影响排序
- [ ] 服务端不能解密消息内容（零知识验证）

---

## 6. 文档更新清单

### CLIENT_ARCHITECTURE.md
- [ ] **第4.1节** - 密钥生成（改为双密钥对）
- [ ] **第3.4节** - AES密钥加密实现
- [ ] **第6节** - 命令处理器（添加`/history`命令）
- [ ] **第8节** - 协议定义（更新注册协议）
- [ ] **第3.6节** - 消息解密流程

### SERVER_ARCHITECTURE.md
- [ ] **第3.2节** - 用户管理器（User结构体改为双公钥）
- [ ] **第3.5.2节** - MessageHandler（实现签名验证）
- [ ] **第3.5节** - 新增3.5.4小节 HistoryHandler
- [ ] **第4节** - 协议定义（添加History相关协议）
- [ ] **第9节** - 优雅关闭（从注释改为完整实现）
- [ ] **第7节** - 安全措施（添加限制实现）

### 终端聊天室需求文档_v2.1.md
- [ ] **第4.2节** - 密钥算法选择（修正Ed25519描述）
- [ ] **第5.2节** - 消息格式（添加双时间戳说明）
- [ ] **第6.3节** - 历史消息查询（补充协议格式）

---

## 7. 依赖库清单

### 客户端 + 服务端共用
- `golang.org/x/crypto/curve25519` - X25519密钥交换
- `golang.org/x/crypto/chacha20poly1305` - AES密钥包装加密
- `golang.org/x/crypto/hkdf` - 密钥派生函数
- `crypto/ed25519` - 签名验证（标准库）
- `crypto/aes` - 消息加密（标准库）
- `crypto/cipher` - GCM模式（标准库）

### 仅服务端
- `golang.org/x/time/rate` - 速率限制
- `golang.org/x/net/websocket` - WebSocket

---

## 8. 风险与缓解措施

### 风险1: X25519密钥交换实现复杂度
**风险**: ECDH + HKDF + ChaCha20-Poly1305 组合复杂，可能有安全漏洞

**缓解**:
- 使用经过验证的库（golang.org/x/crypto）
- 参考标准实现（libsodium的crypto_box）
- 详细的单元测试
- 代码审查重点关注密码学部分

### 风险2: 双密钥对用户体验复杂度
**风险**: 用户需要管理两对密钥，可能混淆

**缓解**:
- CLI工具自动生成两对密钥
- 统一存储在同一目录（~/.chatroom/keys/）
- 文件命名清晰：username_signing.key 和 username_encrypt.key
- 文档中明确说明用途

### 风险3: 协议向后兼容性
**风险**: 修改注册协议可能导致老客户端无法连接

**缓解**:
- v1.0首次发布，无需考虑向后兼容
- 协议中添加version字段为未来扩展做准备

---

## 9. 关键技术决策

### 决策1: 加密方案选择 X25519（而非RSA）
**理由**:
- 性能优势：X25519比RSA-2048快10-20倍
- 密钥长度：32字节公钥 vs RSA的256字节
- 现代化：Curve25519是现代加密标准
- 一致性：与Ed25519同属Curve25519系列
- Go支持：`golang.org/x/crypto/curve25519` 官方支持

### 决策2: 保留双时间戳
**理由**:
- client_timestamp：签名验证需要
- server_timestamp：权威排序和防重放攻击
- 兼容性：客户端老版本可能仍发送client_timestamp

### 决策3: 历史消息采用反向遍历
**理由**:
- v1.0需求：<100用户，消息量不大
- 简单性：文件存储，无需数据库
- 性能：内存排序足够快
- v2.0可优化：引入数据库和索引

---

## 10. 交付物

### 代码
1. **chatroom-server/** - 服务端完整代码
2. **chatroom-client/** - 客户端完整代码
3. **pkg/protocol/** - 共享协议定义
4. **pkg/crypto/** - 共享加密模块

### 文档
1. **IMPLEMENTATION_PLAN.md** - 本文档
2. **DEVELOPMENT_LOG.md** - 开发日志
3. **CLIENT_ARCHITECTURE.md** - 更新的客户端架构
4. **SERVER_ARCHITECTURE.md** - 更新的服务端架构
5. **终端聊天室需求文档_v2.1.md** - 更新的需求文档

### 测试
1. **单元测试** - 覆盖核心加密和协议模块
2. **集成测试** - E2E测试脚本
3. **安全测试** - 伪造签名、时间戳篡改等

---

## 11. 当前状态

**最后更新**: 2026-03-02
**当前阶段**: 第一阶段 - P0修复
**完成度**: 0%

---

## 12. 联系与支持

如有问题，请查阅开发日志：`DEVELOPMENT_LOG.md`
