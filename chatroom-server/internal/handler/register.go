package handler

import (
	"encoding/json"
	"log"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/message"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// RegisterHandler 注册处理器
type RegisterHandler struct {
	router *message.Router
}

// NewRegisterHandler 创建注册处理器
func NewRegisterHandler(router *message.Router) *RegisterHandler {
	return &RegisterHandler{router: router}
}

// Handle 处理注册请求
func (h *RegisterHandler) Handle(client *connection.Client, msg *protocol.Message) error {
	// 解析注册请求
	var req protocol.RegisterRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return err
	}

	log.Printf("Register request: username=%s, algorithm=%s", req.Username, req.Algorithm)

	// 注册用户（处理冲突）
	username, err := h.router.GetUserManager().Register(
		req.Username,
		req.SigningKey,
		req.EncryptionKey,
		req.Algorithm,
	)
	if err != nil {
		return err
	}

	// 绑定用户名到连接
	if err := h.router.GetConnManager().BindUser(client.ID, username); err != nil {
		return err
	}

	// 保存用户数据
	if err := h.router.GetUserManager().Save(h.router.GetStorage()); err != nil {
		log.Printf("Warning: failed to save user data: %v", err)
	}

	// 获取在线用户列表
	onlineUsers := h.router.GetConnManager().GetOnlineUsers()

	// 响应客户端
	resp := protocol.RegisterResponse{
		Type:             "register_response",
		Success:          true,
		AssignedUsername: username,
		OnlineUsers:      onlineUsers,
	}
	data, _ := json.Marshal(resp)
	client.SendChan <- data

	log.Printf("User registered: %s (original: %s)", username, req.Username)

	// 广播用户上线通知
	notification := protocol.UserOnlineNotification{
		Type:          "user_online",
		Username:      username,
		SigningKey:    req.SigningKey,
		EncryptionKey: req.EncryptionKey,
		Algorithm:     req.Algorithm,
	}
	notifyData, _ := json.Marshal(notification)
	h.router.GetConnManager().Broadcast(notifyData)

	return nil
}
