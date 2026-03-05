package message

import (
	"encoding/json"
	"log"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/storage"
	"github.com/xiaowyu/chatroom-server/internal/user"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// Handler 消息处理器接口
type Handler interface {
	Handle(client *connection.Client, msg *protocol.Message) error
}

// Router 消息路由器
type Router struct {
	handlers    map[string]Handler
	connManager *connection.Manager
	userManager *user.Manager
	storage     storage.Storage
}

// NewRouter 创建消息路由器
func NewRouter(cm *connection.Manager, um *user.Manager, s storage.Storage) *Router {
	r := &Router{
		handlers:    make(map[string]Handler),
		connManager: cm,
		userManager: um,
		storage:     s,
	}

	// 注册处理器（稍后实现具体 handler）
	// r.handlers["register"] = &RegisterHandler{r}
	// r.handlers["message"] = &MessageHandler{r}
	// r.handlers["get_pubkeys"] = &PubKeyHandler{r}
	// r.handlers["history"] = &HistoryHandler{r}

	return r
}

// Route 路由消息到对应的处理器
func (r *Router) Route(client *connection.Client, msg *protocol.Message) {
	handler, ok := r.handlers[msg.Type]
	if !ok {
		r.sendError(client, ": "+msg.Type)
		return
	}

	if err := handler.Handle(client, msg); err != nil {
		log.Printf("Handler error for %s: %v", msg.Type, err)
		r.sendError(client, err.Error())
	}
}

// RegisterHandler 注册处理器
func (r *Router) RegisterHandler(msgType string, handler Handler) {
	r.handlers[msgType] = handler
}

// sendError 发送错误消息
func (r *Router) sendError(client *connection.Client, errMsg string) {
	resp := protocol.Message{
		Type:  "error",
		Error: errMsg,
	}
	data, _ := json.Marshal(resp)
	client.SendChan <- data
}

// GetConnManager 获取连接管理器
func (r *Router) GetConnManager() *connection.Manager {
	return r.connManager
}

// GetUserManager 获取用户管理器
func (r *Router) GetUserManager() *user.Manager {
	return r.userManager
}

// GetStorage 获取存储
func (r *Router) GetStorage() storage.Storage {
	return r.storage
}
