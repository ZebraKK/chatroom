# 服务端架构设计文档

## 1. 架构概览

### 1.1 总体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      chatroom-server                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   HTTP/WSS   │  │   Router     │  │   Storage    │     │
│  │   Handler    │──│   Layer      │──│   Engine     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│         │                  │                  │            │
│         │                  │                  │            │
│  ┌──────▼──────┐  ┌────────▼────────┐  ┌─────▼──────┐    │
│  │ Connection  │  │   User Manager  │  │   File     │    │
│  │   Manager   │  │                 │  │   I/O      │    │
│  └─────────────┘  └─────────────────┘  └────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 核心职责

服务端作为**中心转发节点**，负责：
- 🔌 维护 WebSocket 长连接
- 👥 管理用户注册与在线状态
- 🔑 存储和分发用户公钥
- 📮 转发加密消息（不解密）
- 💾 持久化加密消息记录
- ⏱️ 保证消息时间顺序

**关键原则：零知识服务器** - 服务端无法解密任何消息内容

---

## 2. 目录结构

```
chatroom-server/
├── main.go                 # 程序入口
├── cmd/
│   └── server/
│       └── main.go         # 服务启动入口
├── internal/
│   ├── server/
│   │   └── server.go       # HTTP/WebSocket 服务器
│   ├── handler/
│   │   ├── handler.go      # 消息处理器接口
│   │   ├── register.go     # 注册处理
│   │   ├── message.go      # 消息处理
│   │   └── pubkey.go       # 公钥处理
│   ├── connection/
│   │   ├── manager.go      # 连接管理器
│   │   └── client.go       # 客户端连接封装
│   ├── user/
│   │   ├── manager.go      # 用户管理器
│   │   └── user.go         # 用户模型
│   ├── message/
│   │   ├── message.go      # 消息模型
│   │   └── router.go       # 消息路由
│   └── storage/
│       ├── storage.go      # 存储接口
│       ├── file.go         # 文件存储实现
│       └── memory.go       # 内存缓存
├── pkg/
│   ├── protocol/
│   │   └── protocol.go     # 协议定义
│   └── util/
│       └── util.go         # 工具函数
├── data/                   # 数据目录（运行时创建）
│   ├── messages.jsonl      # 消息存储
│   └── users.json          # 用户公钥存储
├── go.mod
└── go.sum
```

---

## 3. 核心模块设计

### 3.1 Server 模块

**职责：** HTTP/WebSocket 服务器启动和管理

```go
// internal/server/server.go

package server

import (
    "net/http"
    "golang.org/x/net/websocket"
)

type Server struct {
    addr          string
    certFile      string
    keyFile       string
    connManager   *connection.Manager
    userManager   *user.Manager
    messageRouter *message.Router
    storage       storage.Storage
}

func New(addr, certFile, keyFile string) *Server {
    return &Server{
        addr:          addr,
        certFile:      certFile,
        keyFile:       keyFile,
        connManager:   connection.NewManager(),
        userManager:   user.NewManager(),
        messageRouter: message.NewRouter(),
        storage:       storage.NewFileStorage("./data"),
    }
}

func (s *Server) Start() error {
    // 加载用户数据
    if err := s.userManager.Load(s.storage); err != nil {
        return err
    }

    // 注册路由
    http.Handle("/ws", websocket.Handler(s.handleWebSocket))
    http.HandleFunc("/health", s.handleHealth)

    // 启动 HTTPS 服务
    return http.ListenAndServeTLS(s.addr, s.certFile, s.keyFile, nil)
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
    // 创建客户端连接
    client := s.connManager.AddClient(ws)
    defer s.connManager.RemoveClient(client.ID)

    // 消息循环
    for {
        var msg protocol.Message
        if err := websocket.JSON.Receive(ws, &msg); err != nil {
            break
        }

        // 路由消息
        s.messageRouter.Route(client, &msg)
    }
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

**关键点：**
- 使用 `golang.org/x/net/websocket`（官方扩展包）
- TLS 加密传输（WSS）
- 每个连接一个 goroutine 处理
- 优雅的连接生命周期管理

---

### 3.2 Connection Manager（连接管理器）

**职责：** 维护所有活跃的 WebSocket 连接

```go
// internal/connection/manager.go

package connection

import (
    "sync"
    "golang.org/x/net/websocket"
)

type Client struct {
    ID       string           // 唯一连接 ID
    Username string           // 用户名（注册后赋值）
    Conn     *websocket.Conn  // WebSocket 连接
    SendChan chan []byte      // 发送消息通道
}

type Manager struct {
    mu      sync.RWMutex
    clients map[string]*Client  // connID -> Client
    users   map[string]*Client  // username -> Client
}

func NewManager() *Manager {
    return &Manager{
        clients: make(map[string]*Client),
        users:   make(map[string]*Client),
    }
}

// 添加连接
func (m *Manager) AddClient(ws *websocket.Conn) *Client {
    m.mu.Lock()
    defer m.mu.Unlock()

    client := &Client{
        ID:       generateID(),
        Conn:     ws,
        SendChan: make(chan []byte, 256),
    }

    m.clients[client.ID] = client

    // 启动发送协程
    go client.writePump()

    return client
}

// 移除连接
func (m *Manager) RemoveClient(id string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    client, ok := m.clients[id]
    if !ok {
        return
    }

    // 清理用户映射
    if client.Username != "" {
        delete(m.users, client.Username)
    }

    // 关闭连接
    close(client.SendChan)
    client.Conn.Close()

    delete(m.clients, id)
}

// 绑定用户名到连接
func (m *Manager) BindUser(connID, username string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    client, ok := m.clients[connID]
    if !ok {
        return
    }

    client.Username = username
    m.users[username] = client
}

// 获取所有在线用户名
func (m *Manager) GetOnlineUsers() []string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    users := make([]string, 0, len(m.users))
    for username := range m.users {
        users = append(users, username)
    }
    return users
}

// 广播消息给所有在线用户
func (m *Manager) Broadcast(data []byte) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    for _, client := range m.users {
        select {
        case client.SendChan <- data:
        default:
            // 通道满，跳过
        }
    }
}

// 发送给特定用户
func (m *Manager) SendToUser(username string, data []byte) error {
    m.mu.RLock()
    client, ok := m.users[username]
    m.mu.RUnlock()

    if !ok {
        return errors.New("user not online")
    }

    select {
    case client.SendChan <- data:
        return nil
    default:
        return errors.New("send buffer full")
    }
}

// 客户端写入协程
func (c *Client) writePump() {
    for data := range c.SendChan {
        if err := websocket.Message.Send(c.Conn, string(data)); err != nil {
            break
        }
    }
}

func generateID() string {
    // 生成唯一 ID（使用 crypto/rand）
    b := make([]byte, 16)
    rand.Read(b)
    return fmt.Sprintf("%x", b)
}
```

**关键设计：**
- 双层映射：`connID -> Client` 和 `username -> Client`
- 异步发送通道，避免阻塞
- 读写分离：读在主循环，写在独立 goroutine
- 线程安全：RWMutex 保护并发访问

---

### 3.3 User Manager（用户管理器）

**职责：** 用户注册、公钥管理、用户名冲突处理

```go
// internal/user/manager.go

package user

import (
    "sync"
    "fmt"
)

type User struct {
    Username     string `json:"username"`
    PublicKey    string `json:"public_key"`     // Base64 编码
    Algorithm    string `json:"algorithm"`      // ed25519/rsa2048
    RegisteredAt int64  `json:"registered_at"`
}

type Manager struct {
    mu    sync.RWMutex
    users map[string]*User  // username -> User
}

func NewManager() *Manager {
    return &Manager{
        users: make(map[string]*User),
    }
}

// 注册用户（处理用户名冲突）
func (m *Manager) Register(username, publicKey, algorithm string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 查找可用用户名
    finalUsername := username
    counter := 1
    for {
        if _, exists := m.users[finalUsername]; !exists {
            break
        }
        finalUsername = fmt.Sprintf("%s_%d", username, counter)
        counter++
    }

    // 创建用户
    user := &User{
        Username:     finalUsername,
        PublicKey:    publicKey,
        Algorithm:    algorithm,
        RegisteredAt: time.Now().Unix(),
    }

    m.users[finalUsername] = user

    return finalUsername, nil
}

// 获取用户公钥
func (m *Manager) GetPublicKey(username string) (string, string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    user, ok := m.users[username]
    if !ok {
        return "", "", errors.New("user not found")
    }

    return user.PublicKey, user.Algorithm, nil
}

// 获取多个用户的公钥
func (m *Manager) GetPublicKeys(usernames []string) map[string]*User {
    m.mu.RLock()
    defer m.mu.RUnlock()

    result := make(map[string]*User)
    for _, username := range usernames {
        if user, ok := m.users[username]; ok {
            result[username] = user
        }
    }

    return result
}

// 获取所有用户
func (m *Manager) GetAllUsers() []*User {
    m.mu.RLock()
    defer m.mu.RUnlock()

    users := make([]*User, 0, len(m.users))
    for _, user := range m.users {
        users = append(users, user)
    }

    return users
}

// 加载用户数据
func (m *Manager) Load(storage storage.Storage) error {
    users, err := storage.LoadUsers()
    if err != nil {
        return err
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    for _, user := range users {
        m.users[user.Username] = user
    }

    return nil
}

// 保存用户数据
func (m *Manager) Save(storage storage.Storage) error {
    m.mu.RLock()
    users := m.GetAllUsers()
    m.mu.RUnlock()

    return storage.SaveUsers(users)
}
```

**关键点：**
- 用户名冲突自动添加后缀（`alice`, `alice_1`, `alice_2`...）
- 公钥和算法类型一起存储
- 支持批量获取公钥（优化性能）
- 持久化到存储层

---

### 3.4 Message Router（消息路由器）

**职责：** 根据消息类型分发到不同的处理器

```go
// internal/message/router.go

package message

import (
    "encoding/json"
)

type Router struct {
    handlers      map[string]Handler
    connManager   *connection.Manager
    userManager   *user.Manager
    storage       storage.Storage
}

type Handler interface {
    Handle(client *connection.Client, msg *protocol.Message) error
}

func NewRouter(cm *connection.Manager, um *user.Manager, s storage.Storage) *Router {
    r := &Router{
        handlers:    make(map[string]Handler),
        connManager: cm,
        userManager: um,
        storage:     s,
    }

    // 注册处理器
    r.handlers["register"] = &RegisterHandler{r}
    r.handlers["message"] = &MessageHandler{r}
    r.handlers["get_pubkeys"] = &PubKeyHandler{r}

    return r
}

func (r *Router) Route(client *connection.Client, msg *protocol.Message) {
    handler, ok := r.handlers[msg.Type]
    if !ok {
        r.sendError(client, "unknown message type")
        return
    }

    if err := handler.Handle(client, msg); err != nil {
        r.sendError(client, err.Error())
    }
}

func (r *Router) sendError(client *connection.Client, errMsg string) {
    resp := protocol.Message{
        Type:  "error",
        Error: errMsg,
    }
    data, _ := json.Marshal(resp)
    client.SendChan <- data
}
```

---

### 3.5 Handler 实现

#### 3.5.1 Register Handler（注册处理器）

```go
// internal/handler/register.go

package handler

type RegisterHandler struct {
    router *message.Router
}

func (h *RegisterHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    // 解析注册请求
    var req protocol.RegisterRequest
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        return err
    }

    // 注册用户（处理冲突）
    username, err := h.router.userManager.Register(
        req.Username,
        req.PublicKey,
        req.Algorithm,
    )
    if err != nil {
        return err
    }

    // 绑定用户名到连接
    h.router.connManager.BindUser(client.ID, username)

    // 保存用户数据
    h.router.userManager.Save(h.router.storage)

    // 获取在线用户列表
    onlineUsers := h.router.connManager.GetOnlineUsers()

    // 响应客户端
    resp := protocol.RegisterResponse{
        Type:             "register_response",
        Success:          true,
        AssignedUsername: username,
        OnlineUsers:      onlineUsers,
    }
    data, _ := json.Marshal(resp)
    client.SendChan <- data

    // 广播用户上线通知
    notification := protocol.UserOnlineNotification{
        Type:      "user_online",
        Username:  username,
        PublicKey: req.PublicKey,
        Algorithm: req.Algorithm,
    }
    notifyData, _ := json.Marshal(notification)
    h.router.connManager.Broadcast(notifyData)

    return nil
}
```

#### 3.5.2 Message Handler（消息处理器）

```go
// internal/handler/message.go

package handler

type MessageHandler struct {
    router *message.Router
}

func (h *MessageHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    // 验证用户已登录
    if client.Username == "" {
        return errors.New("not authenticated")
    }

    // 解析消息
    var chatMsg protocol.ChatMessage
    if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
        return err
    }

    // 验证发送者
    if chatMsg.From != client.Username {
        return errors.New("invalid sender")
    }

    // TODO: 验证签名

    // 保存消息（加密状态）
    if err := h.router.storage.SaveMessage(&chatMsg); err != nil {
        return err
    }

    // 转发给每个接收者
    for _, recipient := range chatMsg.Recipients {
        // 构造单个接收者的消息
        individualMsg := protocol.ChatMessage{
            Type:                "message",
            From:                chatMsg.From,
            Timestamp:           chatMsg.Timestamp,
            AESEncryptedMessage: chatMsg.AESEncryptedMessage,
            EncryptedAESKey:     recipient.EncryptedAESKey,
            Signature:           chatMsg.Signature,
        }

        data, _ := json.Marshal(individualMsg)
        h.router.connManager.SendToUser(recipient.To, data)
    }

    return nil
}
```

#### 3.5.3 PubKey Handler（公钥处理器）

```go
// internal/handler/pubkey.go

package handler

type PubKeyHandler struct {
    router *message.Router
}

func (h *PubKeyHandler) Handle(client *connection.Client, msg *protocol.Message) error {
    var req protocol.PubKeyRequest
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        return err
    }

    // 如果 users 为空，返回所有在线用户的公钥
    usernames := req.Users
    if len(usernames) == 0 {
        usernames = h.router.connManager.GetOnlineUsers()
    }

    // 获取公钥
    keys := h.router.userManager.GetPublicKeys(usernames)

    // 响应
    resp := protocol.PubKeyResponse{
        Type: "pubkeys",
        Keys: keys,
    }
    data, _ := json.Marshal(resp)
    client.SendChan <- data

    return nil
}
```

---

### 3.6 Storage 模块

**职责：** 持久化消息和用户数据

```go
// internal/storage/storage.go

package storage

type Storage interface {
    SaveMessage(msg *protocol.ChatMessage) error
    LoadMessages(limit int) ([]*protocol.ChatMessage, error)
    SaveUsers(users []*user.User) error
    LoadUsers() ([]*user.User, error)
}

// internal/storage/file.go

type FileStorage struct {
    dataDir      string
    messagePath  string
    userPath     string
    mu           sync.Mutex
}

func NewFileStorage(dataDir string) *FileStorage {
    return &FileStorage{
        dataDir:     dataDir,
        messagePath: filepath.Join(dataDir, "messages.jsonl"),
        userPath:    filepath.Join(dataDir, "users.json"),
    }
}

func (s *FileStorage) SaveMessage(msg *protocol.ChatMessage) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // 确保目录存在
    os.MkdirAll(s.dataDir, 0755)

    // 追加到文件
    f, err := os.OpenFile(s.messagePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    data, _ := json.Marshal(msg)
    _, err = f.Write(append(data, '\n'))
    return err
}

func (s *FileStorage) LoadMessages(limit int) ([]*protocol.ChatMessage, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    f, err := os.Open(s.messagePath)
    if err != nil {
        if os.IsNotExist(err) {
            return []*protocol.ChatMessage{}, nil
        }
        return nil, err
    }
    defer f.Close()

    var messages []*protocol.ChatMessage
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var msg protocol.ChatMessage
        if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
            continue
        }
        messages = append(messages, &msg)
    }

    // 返回最后 limit 条
    if limit > 0 && len(messages) > limit {
        messages = messages[len(messages)-limit:]
    }

    return messages, nil
}

func (s *FileStorage) SaveUsers(users []*user.User) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    os.MkdirAll(s.dataDir, 0755)

    data, err := json.MarshalIndent(users, "", "  ")
    if err != nil {
        return err
    }

    return ioutil.WriteFile(s.userPath, data, 0644)
}

func (s *FileStorage) LoadUsers() ([]*user.User, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    data, err := ioutil.ReadFile(s.userPath)
    if err != nil {
        if os.IsNotExist(err) {
            return []*user.User{}, nil
        }
        return nil, err
    }

    var users []*user.User
    if err := json.Unmarshal(data, &users); err != nil {
        return nil, err
    }

    return users, nil
}
```

---

## 4. 协议定义

```go
// pkg/protocol/protocol.go

package protocol

type Message struct {
    Type  string          `json:"type"`
    Data  json.RawMessage `json:"data,omitempty"`
    Error string          `json:"error,omitempty"`
}

type RegisterRequest struct {
    Username  string `json:"username"`
    PublicKey string `json:"public_key"`
    Algorithm string `json:"algorithm"`
}

type RegisterResponse struct {
    Type             string   `json:"type"`
    Success          bool     `json:"success"`
    AssignedUsername string   `json:"assigned_username"`
    OnlineUsers      []string `json:"online_users"`
}

type ChatMessage struct {
    Type                string      `json:"type"`
    From                string      `json:"from"`
    Timestamp           int64       `json:"timestamp"`
    AESEncryptedMessage string      `json:"aes_encrypted_message"`
    Recipients          []Recipient `json:"recipients,omitempty"` // 客户端发送时使用
    EncryptedAESKey     string      `json:"encrypted_aes_key,omitempty"` // 服务器转发时使用
    Signature           string      `json:"signature"`
}

type Recipient struct {
    To              string `json:"to"`
    EncryptedAESKey string `json:"encrypted_aes_key"`
}

type PubKeyRequest struct {
    Type  string   `json:"type"`
    Users []string `json:"users"`
}

type PubKeyResponse struct {
    Type string           `json:"type"`
    Keys map[string]*User `json:"keys"`
}

type UserOnlineNotification struct {
    Type      string `json:"type"`
    Username  string `json:"username"`
    PublicKey string `json:"public_key"`
    Algorithm string `json:"algorithm"`
}

type UserOfflineNotification struct {
    Type     string `json:"type"`
    Username string `json:"username"`
}
```

---

## 5. 并发模型

### 5.1 Goroutine 结构

```
Main Goroutine
    │
    ├─ HTTP Server (Accept Loop)
    │   │
    │   ├─ Client 1 Read Goroutine ──┐
    │   │                             │
    │   ├─ Client 1 Write Goroutine  │
    │   │                             ├─> Router (共享)
    │   ├─ Client 2 Read Goroutine ──┤
    │   │                             │
    │   ├─ Client 2 Write Goroutine  │
    │   │                             │
    │   └─ ...                        │
    │                                 │
    └─ Storage Goroutine (可选异步写入)
```

### 5.2 并发安全策略

1. **连接管理器**：`sync.RWMutex` 保护连接映射
2. **用户管理器**：`sync.RWMutex` 保护用户数据
3. **存储模块**：`sync.Mutex` 保护文件写入
4. **消息发送**：每个客户端独立的 channel，避免竞争

### 5.3 优雅关闭

```go
func (s *Server) Shutdown(ctx context.Context) error {
    // 停止接受新连接
    // 关闭所有现有连接
    // 保存数据
    // 等待所有 goroutine 退出
}
```

---

## 6. 错误处理

### 6.1 错误分类

| 错误类型 | 处理策略 |
|---------|---------|
| 网络错误 | 关闭连接，清理资源 |
| 协议错误 | 返回错误消息，保持连接 |
| 存储错误 | 记录日志，返回失败 |
| 认证错误 | 拒绝请求，可能关闭连接 |

### 6.2 错误响应格式

```json
{
  "type": "error",
  "error": "error message here",
  "code": "AUTH_FAILED"
}
```

---

## 7. 性能优化

### 7.1 内存优化
- ✅ 使用对象池（`sync.Pool`）复用临时对象
- ✅ 限制发送 channel 大小（256 条消息）
- ✅ 定期清理断开的连接

### 7.2 I/O 优化
- ✅ 批量写入消息到磁盘（可选）
- ✅ 使用 buffered I/O
- ✅ 异步存储（不阻塞消息转发）

### 7.3 并发优化
- ✅ 读写分离
- ✅ 细粒度锁（RWMutex）
- ✅ 无锁化的 channel 通信

---

## 8. 安全措施

### 8.1 传输安全
- ✅ 强制 TLS 1.2+
- ✅ 证书验证

### 8.2 应用安全
- ✅ 消息签名验证（防止伪造）
- ✅ 用户名长度限制（防止滥用）
- ✅ 消息大小限制（防止 DoS）
- ✅ 连接速率限制（可选）

### 8.3 数据安全
- ✅ 服务端只存储加密消息
- ✅ 不记录私钥或明文
- ✅ 文件权限控制（0600）

---

## 9. 监控与日志

### 9.1 日志级别
- **INFO**：启动、关闭、用户注册
- **WARN**：连接异常、协议错误
- **ERROR**：存储失败、严重错误

### 9.2 监控指标
- 在线用户数
- 消息吞吐量（msg/s）
- 平均延迟
- 错误率

```go
type Metrics struct {
    OnlineUsers   int64
    TotalMessages int64
    ErrorCount    int64
}
```

---

## 10. 部署配置

### 10.1 命令行参数

```bash
./chatroom-server \
    -addr :8443 \
    -cert ./certs/server.crt \
    -key ./certs/server.key \
    -data ./data
```

### 10.2 环境变量

```bash
export CHATROOM_ADDR=":8443"
export CHATROOM_CERT_FILE="./certs/server.crt"
export CHATROOM_KEY_FILE="./certs/server.key"
export CHATROOM_DATA_DIR="./data"
```

### 10.3 生成自签名证书

```bash
openssl req -x509 -newkey rsa:4096 -keyout server.key \
    -out server.crt -days 365 -nodes \
    -subj "/CN=localhost"
```

---

## 11. 开发计划

### Phase 1: 核心框架（1-2天）
- [x] 项目结构搭建
- [ ] WebSocket 服务器
- [ ] 连接管理器
- [ ] 基础协议定义

### Phase 2: 用户管理（1天）
- [ ] 用户注册
- [ ] 用户名冲突处理
- [ ] 公钥存储与分发

### Phase 3: 消息系统（1-2天）
- [ ] 消息路由
- [ ] 消息转发
- [ ] 消息持久化

### Phase 4: 测试与优化（1天）
- [ ] 单元测试
- [ ] 并发测试
- [ ] 性能测试

---

## 12. 总结

服务端架构设计核心原则：

1. **简洁** - 只做必要的事情
2. **安全** - 零知识服务器，不解密内容
3. **高效** - 异步 I/O，并发处理
4. **可靠** - 错误处理，优雅关闭
5. **可维护** - 清晰的模块划分

**关键技术点：**
- WebSocket 长连接
- Goroutine 并发模型
- Channel 异步通信
- 文件持久化存储
- 端到端加密转发
