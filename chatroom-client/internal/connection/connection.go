package connection

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/xiaowyu/chatroom-client/pkg/protocol"
)

// Connection WebSocket 连接管理器
type Connection struct {
	ws          *websocket.Conn
	url         string
	origin      string
	mu          sync.RWMutex
	connected   bool
	sendChan    chan []byte
	receiveChan chan *protocol.Message
	closeChan   chan struct{}
	onMessage   func(*protocol.Message)
}

// New 创建新连接
func New(serverURL string) *Connection {
	return &Connection{
		url:         serverURL,
		origin:      "https://localhost",
		sendChan:    make(chan []byte, 256),
		receiveChan: make(chan *protocol.Message, 256),
		closeChan:   make(chan struct{}),
	}
}

// Connect 连接到服务器
func (c *Connection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return errors.New("already connected")
	}

	// WebSocket 配置（跳过 TLS 验证，仅用于测试）
	config, err := websocket.NewConfig(c.url, c.origin)
	if err != nil {
		return err
	}
	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true, // 仅用于测试自签名证书
	}

	// 连接
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return err
	}

	c.ws = ws
	c.connected = true

	// 启动读写协程
	go c.readPump()
	go c.writePump()

	log.Println("✅ Connected to server")
	return nil
}

// Disconnect 断开连接
func (c *Connection) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	close(c.closeChan)
	c.ws.Close()
	c.connected = false

	log.Println("👋 Disconnected from server")
}

// SendMessage 发送消息（自动包装为 Message 格式）
func (c *Connection) SendMessage(msg interface{}) error {
	// 根据消息类型自动推断 type 字段
	var msgType string
	var msgData interface{}

	switch v := msg.(type) {
	case protocol.RegisterRequest:
		msgType = "register"
		msgData = v
	case protocol.ChatMessage:
		msgType = "message"
		msgData = v
	case *protocol.ChatMessage:
		msgType = "message"
		msgData = v
	case protocol.PubKeyRequest:
		msgType = "get_pubkeys"
		msgData = v
	case protocol.HistoryRequest:
		msgType = "history"
		msgData = v
	case protocol.Message:
		// 如果已经是 Message 类型，直接使用
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		select {
		case c.sendChan <- data:
			return nil
		case <-time.After(5 * time.Second):
			return errors.New("send timeout")
		}
	default:
		return errors.New("unknown message type")
	}

	// 序列化消息数据
	dataBytes, err := json.Marshal(msgData)
	if err != nil {
		return err
	}

	// 包装成 Message 格式
	envelope := protocol.Message{
		Type: msgType,
		Data: json.RawMessage(dataBytes),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	select {
	case c.sendChan <- data:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("send timeout")
	}
}

// ReceiveMessage 接收消息（阻塞）
func (c *Connection) ReceiveMessage() (*protocol.Message, error) {
	select {
	case msg := <-c.receiveChan:
		return msg, nil
	case <-c.closeChan:
		return nil, errors.New("connection closed")
	}
}

// SetMessageHandler 设置消息处理回调
func (c *Connection) SetMessageHandler(handler func(*protocol.Message)) {
	c.onMessage = handler
}

// readPump 读取消息循环
func (c *Connection) readPump() {
	defer func() {
		c.Disconnect()
	}()

	for {
		var msg protocol.Message
		if err := websocket.JSON.Receive(c.ws, &msg); err != nil {
			if c.isConnected() {
				log.Printf("❌ Read error: %v", err)
			}
			return
		}

		// 调用回调或放入通道
		if c.onMessage != nil {
			c.onMessage(&msg)
		} else {
			select {
			case c.receiveChan <- &msg:
			default:
				log.Println("⚠️  Receive buffer full, dropping message")
			}
		}
	}
}

// writePump 发送消息循环
func (c *Connection) writePump() {
	for {
		select {
		case data := <-c.sendChan:
			if err := websocket.Message.Send(c.ws, string(data)); err != nil {
				log.Printf("❌ Write error: %v", err)
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

// isConnected 检查是否已连接
func (c *Connection) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// IsConnected 公开的连接状态检查
func (c *Connection) IsConnected() bool {
	return c.isConnected()
}
