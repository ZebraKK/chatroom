package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/handler"
	"github.com/xiaowyu/chatroom-server/internal/message"
	"github.com/xiaowyu/chatroom-server/internal/storage"
	"github.com/xiaowyu/chatroom-server/internal/user"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// 安全限制常量（P1功能）
const (
	MaxConnections  = 100
	MaxMessageSize  = 64 * 1024 // 64KB
	RateLimitPerSec = 10
)

// Server HTTP/WebSocket 服务器
type Server struct {
	addr          string
	certFile      string
	keyFile       string
	connManager   *connection.Manager
	userManager   *user.Manager
	messageRouter *message.Router
	storage       storage.Storage
	connLimiter   *ConnectionLimiter
	shutdown      chan struct{}
	wg            sync.WaitGroup
	httpServer    *http.Server
}

// ConnectionLimiter 连接限制器（P1功能）
type ConnectionLimiter struct {
	mu    sync.Mutex
	count int
}

// New 创建服务器
func New(addr, certFile, keyFile, dataDir string) *Server {
	s := &Server{
		addr:        addr,
		certFile:    certFile,
		keyFile:     keyFile,
		connManager: connection.NewManager(),
		userManager: user.NewManager(),
		storage:     storage.NewFileStorage(dataDir),
		connLimiter: &ConnectionLimiter{},
		shutdown:    make(chan struct{}),
	}

	// 创建消息路由器
	s.messageRouter = message.NewRouter(s.connManager, s.userManager, s.storage)

	// 注册处理器
	s.messageRouter.RegisterHandler("register", handler.NewRegisterHandler(s.messageRouter))
	s.messageRouter.RegisterHandler("message", handler.NewMessageHandler(s.messageRouter))
	s.messageRouter.RegisterHandler("get_pubkeys", handler.NewPubKeyHandler(s.messageRouter))
	s.messageRouter.RegisterHandler("history", handler.NewHistoryHandler(s.messageRouter))

	return s
}

// Start 启动服务器
func (s *Server) Start() error {
	// 加载用户数据
	if err := s.userManager.Load(s.storage); err != nil {
		log.Printf("Warning: failed to load users: %v", err)
	} else {
		log.Printf("Loaded %d users from storage", len(s.userManager.GetAllUsers()))
	}

	// 注册路由
	http.Handle("/ws", websocket.Handler(s.handleWebSocket))
	http.HandleFunc("/health", s.handleHealth)

	// 创建 HTTP 服务器
	s.httpServer = &http.Server{
		Addr:         s.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 注册系统信号处理（优雅关闭，P1功能）
	s.setupSignalHandler()

	log.Printf("🚀 Server starting on %s", s.addr)
	log.Printf("📁 Data directory: %s", s.storage.(*storage.FileStorage))
	log.Printf("🔒 TLS cert: %s", s.certFile)

	// 启动 HTTPS 服务
	return s.httpServer.ListenAndServeTLS(s.certFile, s.keyFile)
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(ws *websocket.Conn) {
	// 1. 检查连接限制（P1功能）
	if err := s.connLimiter.Acquire(); err != nil {
		log.Printf("❌ Connection rejected: %v", err)
		ws.Close()
		return
	}
	defer s.connLimiter.Release()

	// 2. 设置消息大小限制（P1功能）
	ws.MaxPayloadBytes = MaxMessageSize

	// 3. 创建客户端连接
	client := s.connManager.AddClient(ws)
	defer func() {
		s.connManager.RemoveClient(client.ID)
		log.Printf("👋 Client disconnected: id=%s, username=%s", client.ID, client.Username)

		// 广播用户下线通知
		if client.Username != "" {
			notification := protocol.UserOfflineNotification{
				Type:     "user_offline",
				Username: client.Username,
			}
			data, _ := json.Marshal(notification)
			s.connManager.Broadcast(data)
		}
	}()

	log.Printf("✅ Client connected: id=%s", client.ID)

	// 4. 创建速率限制器（P1功能）
	limiter := rate.NewLimiter(rate.Limit(RateLimitPerSec), RateLimitPerSec*2)

	// 5. 消息循环
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

		// 路由消息
		s.messageRouter.Route(client, &msg)
	}
}

// handleHealth 健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// setupSignalHandler 设置信号处理（优雅关闭，P1功能）
func (s *Server) setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("📢 Received shutdown signal")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.Shutdown(ctx)
		os.Exit(0)
	}()
}

// Shutdown 优雅关闭（P1功能）
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("🛑 Starting graceful shutdown...")

	// 1. 停止接受新连接
	close(s.shutdown)

	// 2. 广播服务器关闭通知
	notification := protocol.ServerShutdownNotification{
		Type:    "server_shutdown",
		Message: "服务器正在关闭",
	}
	data, _ := json.Marshal(notification)
	s.connManager.Broadcast(data)

	// 3. 等待消息发送完成
	time.Sleep(1 * time.Second)

	// 4. 关闭所有 WebSocket 连接
	s.connManager.CloseAll()

	// 5. 保存用户数据
	if err := s.userManager.Save(s.storage); err != nil {
		log.Printf("❌ Failed to save user data: %v", err)
	} else {
		log.Println("✅ User data saved")
	}

	// 6. 关闭 HTTP 服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("❌ HTTP server shutdown error: %v", err)
		return err
	}

	log.Println("✅ Graceful shutdown completed")
	return nil
}

// Acquire 获取连接许可（P1功能）
func (l *ConnectionLimiter) Acquire() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.count >= MaxConnections {
		return &LimitError{Message: "server at capacity"}
	}

	l.count++
	return nil
}

// Release 释放连接许可（P1功能）
func (l *ConnectionLimiter) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.count--
}

// LimitError 限制错误
type LimitError struct {
	Message string
}

func (e *LimitError) Error() string {
	return e.Message
}
