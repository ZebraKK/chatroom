客户端核心职责                                                                                                                                           │
│                                                                                                                                                          │
│ 根据需求文档，客户端需要实现：                                                                                                                           │
│                                                                                                                                                          │
│ 1. 密钥管理                                                                                                                                              │
│   - 首次运行时生成 Ed25519/RSA 密钥对                                                                                                                    │
│   - 私钥安全存储在本地（~/.chatroom/keys/）                                                                                                              │
│   - 公钥上传到服务器                                                                                                                                     │
│ 2. WebSocket 通信                                                                                                                                        │
│   - 建立和维护与服务器的 WSS 长连接                                                                                                                      │
│   - 处理断线重连                                                                                                                                         │
│   - 发送/接收 JSON 格式消息                                                                                                                              │
│ 3. 加密引擎                                                                                                                                              │
│   - 发送消息：生成 AES-256 随机密钥 → AES-GCM 加密消息 → 用所有在线用户公钥加密 AES 密钥                                                                 │
│   - 接收消息：用自己的私钥解密 AES 密钥 → 用 AES 密钥解密消息内容                                                                                        │
│   - 消息签名与验证                                                                                                                                       │
│ 4. 终端 UI                                                                                                                                               │
│   - 实时滚动显示消息                                                                                                                                     │
│   - 底部输入区                                                                                                                                           │
│   - 在线用户列表                                                                                                                                         │
│   - 命令处理（/help, /users, /history, /quit 等）                                                                                                        │
│ 5. 配置管理                                                                                                                                              │
│   - 首次使用交互式配置                                                                                                                                   │
│   - 配置文件存储（~/.chatroom/config.json）                                                                                                              │
│   - 后续启动自动加载配置                                                                                                                                 │
│                                                                                                                                                          │
│ 架构设计方案                                                                                                                                             │
│                                                                                                                                                          │
│ 1. 目录结构                                                                                                                                              │
│                                                                                                                                                          │
│ chatroom-client/                                                                                                                                         │
│ ├── main.go                                                                                                                                              │
│ ├── cmd/                                                                                                                                                 │
│ │   └── client/                                                                                                                                          │
│ │       └── main.go                                                                                                                                      │
│ ├── internal/                                                                                                                                            │
│ │   ├── client/                                                                                                                                          │
│ │   │   └── client.go          # 客户端主控制器                                                                                                          │
│ │   ├── crypto/                                                                                                                                          │
│ │   │   ├── keypair.go          # 密钥对生成与管理                                                                                                       │
│ │   │   ├── encrypt.go          # 消息加密                                                                                                               │
│ │   │   ├── decrypt.go          # 消息解密                                                                                                               │
│ │   │   └── signature.go        # 签名与验证                                                                                                             │
│ │   ├── connection/                                                                                                                                      │
│ │   │   ├── websocket.go        # WebSocket 连接管理                                                                                                     │
│ │   │   └── reconnect.go        # 断线重连逻辑                                                                                                           │
│ │   ├── ui/                                                                                                                                              │
│ │   │   ├── terminal.go         # 终端 UI 主控制器                                                                                                       │
│ │   │   ├── input.go            # 输入处理                                                                                                               │
│ │   │   ├── display.go          # 消息显示                                                                                                               │
│ │   │   └── command.go          # 命令解析                                                                                                               │
│ │   ├── message/                                                                                                                                         │
│ │   │   ├── handler.go          # 消息处理器                                                                                                             │
│ │   │   └── queue.go            # 消息队列                                                                                                               │
│ │   └── config/                                                                                                                                          │
│ │       ├── config.go           # 配置管理                                                                                                               │
│ │       └── setup.go            # 首次配置向导                                                                                                           │
│ ├── pkg/                                                                                                                                                 │
│ │   ├── protocol/                                                                                                                                        │
│ │   │   └── protocol.go         # 协议定义（与服务端共享）                                                                                               │
│ │   └── util/                                                                                                                                            │
│ │       └── util.go                                                                                                                                      │
│ ├── go.mod                                                                                                                                               │
│ └── go.sum                                                                                                                                               │
│                                                                                                                                                          │
│ 2. 核心模块设计                                                                                                                                          │
│                                                                                                                                                          │
│ 2.1 Client（客户端主控制器）                                                                                                                             │
│                                                                                                                                                          │
│ type Client struct {                                                                                                                                     │
│     config      *config.Config                                                                                                                           │
│     conn        *connection.WebSocket                                                                                                                    │
│     crypto      *crypto.Manager                                                                                                                          │
│     ui          *ui.Terminal                                                                                                                             │
│     msgHandler  *message.Handler                                                                                                                         │
│                                                                                                                                                          │
│     username    string                                                                                                                                   │
│     onlineUsers map[string]*protocol.User                                                                                                                │
│                                                                                                                                                          │
│     // Channels                                                                                                                                          │
│     sendChan    chan *protocol.Message                                                                                                                   │
│     recvChan    chan *protocol.Message                                                                                                                   │
│     stopChan    chan struct{}                                                                                                                            │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 职责：                                                                                                                                                   │
│ - 协调各模块工作                                                                                                                                         │
│ - 管理应用生命周期                                                                                                                                       │
│ - 处理用户上线/下线通知                                                                                                                                  │
│                                                                                                                                                          │
│ 2.2 Crypto Manager（加密管理器）                                                                                                                         │
│                                                                                                                                                          │
│ type Manager struct {                                                                                                                                    │
│     algorithm   string              // "ed25519" or "rsa2048"                                                                                            │
│     privateKey  crypto.PrivateKey   // 私钥                                                                                                              │
│     publicKey   crypto.PublicKey    // 公钥                                                                                                              │
│                                                                                                                                                          │
│     // 其他用户的公钥缓存                                                                                                                                │
│     peerKeys    map[string]*PeerKey                                                                                                                      │
│     mu          sync.RWMutex                                                                                                                             │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ type PeerKey struct {                                                                                                                                    │
│     Username  string                                                                                                                                     │
│     PublicKey crypto.PublicKey                                                                                                                           │
│     Algorithm string                                                                                                                                     │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 核心方法：                                                                                                                                               │
│ - GenerateKeyPair(algorithm string) - 生成密钥对                                                                                                         │
│ - LoadPrivateKey(path string) - 加载私钥                                                                                                                 │
│ - SavePrivateKey(path string) - 保存私钥                                                                                                                 │
│ - EncryptMessage(plaintext string, recipients []string) - 加密消息                                                                                       │
│ - DecryptMessage(encrypted *EncryptedMessage) - 解密消息                                                                                                 │
│ - SignMessage(message []byte) - 签名                                                                                                                     │
│ - VerifySignature(message, signature []byte, publicKey) - 验证签名                                                                                       │
│                                                                                                                                                          │
│ 2.3 WebSocket Connection                                                                                                                                 │
│                                                                                                                                                          │
│ type WebSocket struct {                                                                                                                                  │
│     url         string                                                                                                                                   │
│     conn        *websocket.Conn                                                                                                                          │
│                                                                                                                                                          │
│     sendQueue   chan []byte                                                                                                                              │
│     recvQueue   chan []byte                                                                                                                              │
│                                                                                                                                                          │
│     reconnect   *ReconnectManager                                                                                                                        │
│                                                                                                                                                          │
│     connected   atomic.Bool                                                                                                                              │
│     stopChan    chan struct{}                                                                                                                            │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 职责：                                                                                                                                                   │
│ - 建立 WSS 连接                                                                                                                                          │
│ - 维护心跳                                                                                                                                               │
│ - 自动重连                                                                                                                                               │
│ - 异步收发消息                                                                                                                                           │
│                                                                                                                                                          │
│ 2.4 Terminal UI                                                                                                                                          │
│                                                                                                                                                          │
│ 使用第三方库：github.com/charmbracelet/bubbletea 或自实现                                                                                                │
│                                                                                                                                                          │
│ type Terminal struct {                                                                                                                                   │
│     // 显示区域                                                                                                                                          │
│     messageArea  *MessageArea                                                                                                                            │
│     inputArea    *InputArea                                                                                                                              │
│     statusBar    *StatusBar                                                                                                                              │
│                                                                                                                                                          │
│     // 状态                                                                                                                                              │
│     messages     []*DisplayMessage                                                                                                                       │
│     inputBuffer  string                                                                                                                                  │
│                                                                                                                                                          │
│     // 渲染                                                                                                                                              │
│     width        int                                                                                                                                     │
│     height       int                                                                                                                                     │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 功能：                                                                                                                                                   │
│ - 滚动消息显示区                                                                                                                                         │
│ - 底部输入区（支持多行）                                                                                                                                 │
│ - 顶部状态栏（在线用户、加密方式）                                                                                                                       │
│ - 命令自动补全                                                                                                                                           │
│ - 颜色高亮                                                                                                                                               │
│                                                                                                                                                          │
│ 2.5 Message Handler                                                                                                                                      │
│                                                                                                                                                          │
│ type Handler struct {                                                                                                                                    │
│     client      *Client                                                                                                                                  │
│     handlers    map[string]MessageHandler                                                                                                                │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ type MessageHandler interface {                                                                                                                          │
│     Handle(msg *protocol.Message) error                                                                                                                  │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 处理器类型：                                                                                                                                             │
│ - RegisterResponseHandler - 注册响应                                                                                                                     │
│ - ChatMessageHandler - 聊天消息                                                                                                                          │
│ - PubKeyResponseHandler - 公钥响应                                                                                                                       │
│ - UserOnlineHandler - 用户上线通知                                                                                                                       │
│ - UserOfflineHandler - 用户下线通知                                                                                                                      │
│                                                                                                                                                          │
│ 3. 消息流程                                                                                                                                              │
│                                                                                                                                                          │
│ 3.1 发送消息流程                                                                                                                                         │
│                                                                                                                                                          │
│ 用户输入 → InputArea                                                                                                                                     │
│     ↓                                                                                                                                                    │
│ 命令判断 (/ 开头)                                                                                                                                        │
│     ↓                                                                                                                                                    │
│ 普通消息 → 加密引擎                                                                                                                                      │
│     ↓                                                                                                                                                    │
│ 生成 AES-256 密钥                                                                                                                                        │
│     ↓                                                                                                                                                    │
│ AES-GCM 加密消息                                                                                                                                         │
│     ↓                                                                                                                                                    │
│ 遍历在线用户公钥                                                                                                                                         │
│     ↓                                                                                                                                                    │
│ 用每个公钥加密 AES 密钥                                                                                                                                  │
│     ↓                                                                                                                                                    │
│ 签名整个消息                                                                                                                                             │
│     ↓                                                                                                                                                    │
│ 封装 JSON → WebSocket 发送                                                                                                                               │
│                                                                                                                                                          │
│ 3.2 接收消息流程                                                                                                                                         │
│                                                                                                                                                          │
│ WebSocket 接收 JSON                                                                                                                                      │
│     ↓                                                                                                                                                    │
│ 消息类型路由                                                                                                                                             │
│     ↓                                                                                                                                                    │
│ ChatMessage → 解密引擎                                                                                                                                   │
│     ↓                                                                                                                                                    │
│ 验证发送者签名                                                                                                                                           │
│     ↓                                                                                                                                                    │
│ 用自己私钥解密 AES 密钥                                                                                                                                  │
│     ↓                                                                                                                                                    │
│ 用 AES 密钥解密消息内容                                                                                                                                  │
│     ↓                                                                                                                                                    │
│ 更新 UI 显示                                                                                                                                             │
│                                                                                                                                                          │
│ 4. 加密实现细节                                                                                                                                          │
│                                                                                                                                                          │
│ 4.1 密钥生成（Ed25519）                                                                                                                                  │
│                                                                                                                                                          │
│ func GenerateEd25519KeyPair() (crypto.PrivateKey, crypto.PublicKey, error) {                                                                             │
│     publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)                                                                                       │
│     if err != nil {                                                                                                                                      │
│         return nil, nil, err                                                                                                                             │
│     }                                                                                                                                                    │
│     return privateKey, publicKey, nil                                                                                                                    │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 4.2 混合加密流程                                                                                                                                         │
│                                                                                                                                                          │
│ // 发送消息加密                                                                                                                                          │
│ func (m *Manager) EncryptMessage(plaintext string, recipients []*PeerKey) (*EncryptedMessage, error) {                                                   │
│     // 1. 生成随机 AES-256 密钥                                                                                                                          │
│     aesKey := make([]byte, 32)                                                                                                                           │
│     rand.Read(aesKey)                                                                                                                                    │
│                                                                                                                                                          │
│     // 2. AES-GCM 加密消息                                                                                                                               │
│     block, _ := aes.NewCipher(aesKey)                                                                                                                    │
│     gcm, _ := cipher.NewGCM(block)                                                                                                                       │
│                                                                                                                                                          │
│     nonce := make([]byte, gcm.NonceSize())                                                                                                               │
│     rand.Read(nonce)                                                                                                                                     │
│                                                                                                                                                          │
│     ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)                                                                                         │
│                                                                                                                                                          │
│     // 3. 用每个接收者公钥加密 AES 密钥                                                                                                                  │
│     encryptedKeys := make([]EncryptedKey, 0, len(recipients))                                                                                            │
│     for _, peer := range recipients {                                                                                                                    │
│         encKey, err := m.encryptAESKey(aesKey, peer.PublicKey)                                                                                           │
│         if err != nil {                                                                                                                                  │
│             continue                                                                                                                                     │
│         }                                                                                                                                                │
│         encryptedKeys = append(encryptedKeys, EncryptedKey{                                                                                              │
│             To:  peer.Username,                                                                                                                          │
│             Key: base64.StdEncoding.EncodeToString(encKey),                                                                                              │
│         })                                                                                                                                               │
│     }                                                                                                                                                    │
│                                                                                                                                                          │
│     return &EncryptedMessage{                                                                                                                            │
│         Ciphertext:    base64.StdEncoding.EncodeToString(ciphertext),                                                                                    │
│         EncryptedKeys: encryptedKeys,                                                                                                                    │
│     }, nil                                                                                                                                               │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ // 接收消息解密                                                                                                                                          │
│ func (m *Manager) DecryptMessage(encrypted *EncryptedMessage, encryptedAESKey string) (string, error) {                                                  │
│     // 1. 用私钥解密 AES 密钥                                                                                                                            │
│     aesKey, err := m.decryptAESKey(encryptedAESKey)                                                                                                      │
│     if err != nil {                                                                                                                                      │
│         return "", err                                                                                                                                   │
│     }                                                                                                                                                    │
│                                                                                                                                                          │
│     // 2. 用 AES 密钥解密消息                                                                                                                            │
│     ciphertext, _ := base64.StdEncoding.DecodeString(encrypted.Ciphertext)                                                                               │
│                                                                                                                                                          │
│     block, _ := aes.NewCipher(aesKey)                                                                                                                    │
│     gcm, _ := cipher.NewGCM(block)                                                                                                                       │
│                                                                                                                                                          │
│     nonceSize := gcm.NonceSize()                                                                                                                         │
│     nonce := ciphertext[:nonceSize]                                                                                                                      │
│     ciphertext = ciphertext[nonceSize:]                                                                                                                  │
│                                                                                                                                                          │
│     plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)                                                                                              │
│     if err != nil {                                                                                                                                      │
│         return "", err                                                                                                                                   │
│     }                                                                                                                                                    │
│                                                                                                                                                          │
│     return string(plaintext), nil                                                                                                                        │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 5. 终端 UI 实现                                                                                                                                          │
│                                                                                                                                                          │
│ 5.1 布局设计                                                                                                                                             │
│                                                                                                                                                          │
│ ┌─────────────────────────────────────────────────────────┐                                                                                              │
│ │ 终端聊天室 | 在线: alice, bob, charlie (3) | Ed25519  │  ← 状态栏                                                                                      │
│ ├─────────────────────────────────────────────────────────┤                                                                                              │
│ │ [10:23:45] alice: 大家好                               │                                                                                               │
│ │ [10:24:12] bob: 早上好                                 │                                                                                               │
│ │ [10:25:33] charlie: 今天讨论什么技术问题？             │  ← 消息区                                                                                     │
│ │ [10:26:01] alice: 我想聊聊 Golang 的并发              │  （滚动）                                                                                      │
│ │                                                         │                                                                                              │
│ │                                                         │                                                                                              │
│ ├─────────────────────────────────────────────────────────┤                                                                                              │
│ │ > 输入你的消息...                                      │  ← 输入区                                                                                     │
│ └─────────────────────────────────────────────────────────┘                                                                                              │
│                                                                                                                                                          │
│ 5.2 使用库选择                                                                                                                                           │
│                                                                                                                                                          │
│ 方案 1：自实现（推荐）                                                                                                                                   │
│ - 使用 golang.org/x/term 处理终端控制                                                                                                                    │
│ - 简单、可控、无外部依赖                                                                                                                                 │
│                                                                                                                                                          │
│ 方案 2：Bubble Tea                                                                                                                                       │
│ - github.com/charmbracelet/bubbletea - TUI 框架                                                                                                          │
│ - 功能强大但增加依赖                                                                                                                                     │
│                                                                                                                                                          │
│ 推荐方案 1，保持技术栈纯净。                                                                                                                             │
│                                                                                                                                                          │
│ 6. 配置管理                                                                                                                                              │
│                                                                                                                                                          │
│ 6.1 配置文件格式                                                                                                                                         │
│                                                                                                                                                          │
│ {                                                                                                                                                        │
│   "username": "alice",                                                                                                                                   │
│   "server": "wss://localhost:8443",                                                                                                                      │
│   "algorithm": "ed25519",                                                                                                                                │
│   "key_path": "/Users/alice/.chatroom/keys/alice.key",                                                                                                   │
│   "auto_reconnect": true,                                                                                                                                │
│   "history_limit": 100                                                                                                                                   │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 6.2 首次配置流程                                                                                                                                         │
│                                                                                                                                                          │
│ func (s *Setup) Run() (*config.Config, error) {                                                                                                          │
│     fmt.Println("欢迎使用终端聊天室！")                                                                                                                  │
│     fmt.Println("检测到首次使用，开始初始化...\n")                                                                                                       │
│                                                                                                                                                          │
│     // 1. 输入用户名                                                                                                                                     │
│     username := s.promptUsername()                                                                                                                       │
│                                                                                                                                                          │
│     // 2. 服务器地址                                                                                                                                     │
│     server := s.promptServer()                                                                                                                           │
│                                                                                                                                                          │
│     // 3. 加密算法                                                                                                                                       │
│     algorithm := s.promptAlgorithm()                                                                                                                     │
│                                                                                                                                                          │
│     // 4. 生成密钥对                                                                                                                                     │
│     fmt.Println("\n生成密钥对中...")                                                                                                                     │
│     keyPath, err := s.generateKeys(username, algorithm)                                                                                                  │
│     if err != nil {                                                                                                                                      │
│         return nil, err                                                                                                                                  │
│     }                                                                                                                                                    │
│     fmt.Printf("✓ 密钥已保存到 %s\n", keyPath)                                                                                                           │
│                                                                                                                                                          │
│     // 5. 保存配置                                                                                                                                       │
│     cfg := &config.Config{                                                                                                                               │
│         Username:  username,                                                                                                                             │
│         Server:    server,                                                                                                                               │
│         Algorithm: algorithm,                                                                                                                            │
│         KeyPath:   keyPath,                                                                                                                              │
│     }                                                                                                                                                    │
│                                                                                                                                                          │
│     if err := cfg.Save(); err != nil {                                                                                                                   │
│         return nil, err                                                                                                                                  │
│     }                                                                                                                                                    │
│                                                                                                                                                          │
│     return cfg, nil                                                                                                                                      │
│ }                                                                                                                                                        │
│                                                                                                                                                          │
│ 7. 并发模型                                                                                                                                              │
│                                                                                                                                                          │
│ Main Goroutine (UI Event Loop)                                                                                                                           │
│     │                                                                                                                                                    │
│     ├─ WebSocket Read Goroutine ────→ recvChan                                                                                                           │
│     │                                                                                                                                                    │
│     ├─ WebSocket Write Goroutine ───← sendChan                                                                                                           │
│     │                                                                                                                                                    │
│     ├─ Message Handler Goroutine                                                                                                                         │
│     │   │                                                                                                                                                │
│     │   └─ 从 recvChan 读取                                                                                                                              │
│     │       ↓                                                                                                                                            │
│     │      解密 → 更新 UI                                                                                                                                │
│     │                                                                                                                                                    │
│     └─ Input Handler Goroutine                                                                                                                           │
│         │                                                                                                                                                │
│         └─ 监听用户输入                                                                                                                                  │
│             ↓                                                                                                                                            │
│            加密 → sendChan                                                                                                                               │
│                                                                                                                                                          │
│ 8. 错误处理                                                                                                                                              │
│                                                                                                                                                          │
│ - 网络错误：自动重连（指数退避）                                                                                                                         │
│ - 解密失败：显示错误提示，跳过该消息                                                                                                                     │
│ - 签名验证失败：警告用户，可能是伪造消息                                                                                                                 │
│ - 配置错误：引导用户重新配置                                                                                                                             │
│                                                                                                                                                          │
│ 9. 关键实现要点                                                                                                                                          │
│                                                                                                                                                          │
│ 1. 密钥安全                                                                                                                                              │
│   - 私钥文件权限设置为 0600                                                                                                                              │
│   - 内存中的私钥使用完立即清零                                                                                                                           │
│   - 不记录日志中的敏感信息                                                                                                                               │
│ 2. 消息顺序                                                                                                                                              │
│   - 按服务器时间戳排序显示                                                                                                                               │
│   - 本地发送的消息立即显示（乐观更新）                                                                                                                   │
│ 3. 性能优化                                                                                                                                              │
│   - 公钥缓存（避免重复加密）                                                                                                                             │
│   - 消息批量渲染                                                                                                                                         │
│   - 限制内存中消息数量                                                                                                                                   │
│ 4. 用户体验                                                                                                                                              │
│   - 平滑滚动                                                                                                                                             │
│   - 输入历史记录（上下箭头）                                                                                                                             │
│   - 命令自动补全                                                                                                                                         │
│   - 颜色区分不同用户                                                                                                                                     │
│                                                                                                                                                          │
│ 10. 文档结构                                                                                                                                             │
│                                                                                                                                                          │
│ 客户端架构设计文档将包含以下章节：                                                                                                                       │
│                                                                                                                                                          │
│ 1. 架构概览 - 总体架构图、核心职责                                                                                                                       │
│ 2. 目录结构 - 完整的项目结构                                                                                                                             │
│ 3. 核心模块设计 - 各模块详细设计                                                                                                                         │
│   - Client 主控制器                                                                                                                                      │
│   - Crypto Manager                                                                                                                                       │
│   - WebSocket Connection                                                                                                                                 │
│   - Terminal UI                                                                                                                                          │
│   - Message Handler                                                                                                                                      │
│   - Config Manager                                                                                                                                       │
│ 4. 加密实现 - 混合加密方案详解                                                                                                                           │
│ 5. 终端 UI 设计 - 布局、交互、渲染                                                                                                                       │
│ 6. 消息流程 - 发送/接收完整流程                                                                                                                          │
│ 7. 并发模型 - Goroutine 结构、通道通信                                                                                                                   │
│ 8. 配置管理 - 首次配置、配置加载                                                                                                                         │
│ 9. 错误处理 - 分类处理策略                                                                                                                               │
│ 10. 安全措施 - 密钥安全、内存安全                                                                                                                        │
│ 11. 开发计划 - 分阶段实现计划                                                                                                                            │
│ 12. 总结 - 设计原则、关键技术
