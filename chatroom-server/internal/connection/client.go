package connection

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/net/websocket"
)

// Client 客户端连接封装
type Client struct {
	ID       string           // 唯一连接 ID
	Username string           // 用户名（注册后赋值）
	Conn     *websocket.Conn  // WebSocket 连接
	SendChan chan []byte      // 发送消息通道
}

// NewClient 创建新客户端
func NewClient(ws *websocket.Conn) *Client {
	return &Client{
		ID:       generateID(),
		Conn:     ws,
		SendChan: make(chan []byte, 256),
	}
}

// WritePump 客户端写入协程
func (c *Client) WritePump() {
	defer c.Conn.Close()

	for data := range c.SendChan {
		if err := websocket.Message.Send(c.Conn, string(data)); err != nil {
			break
		}
	}
}

// generateID 生成唯一连接 ID
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
