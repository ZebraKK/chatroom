package handler

import (
	"encoding/json"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/message"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// PubKeyHandler 公钥处理器
type PubKeyHandler struct {
	router *message.Router
}

// NewPubKeyHandler 创建公钥处理器
func NewPubKeyHandler(router *message.Router) *PubKeyHandler {
	return &PubKeyHandler{router: router}
}

// Handle 处理公钥请求
func (h *PubKeyHandler) Handle(client *connection.Client, msg *protocol.Message) error {
	var req protocol.PubKeyRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return err
	}

	// 如果 users 为空，返回所有在线用户的公钥
	usernames := req.Users
	if len(usernames) == 0 {
		usernames = h.router.GetConnManager().GetOnlineUsers()
	}

	// 获取公钥
	users := h.router.GetUserManager().GetMultiplePublicKeys(usernames)

	// 转换为协议格式
	keys := make(map[string]*protocol.UserPublicKeys)
	for username, user := range users {
		keys[username] = &protocol.UserPublicKeys{
			SigningKey:    user.SigningKey,
			EncryptionKey: user.EncryptionKey,
			Algorithm:     user.Algorithm,
		}
	}

	// 响应（包装成 Message 格式）
	resp := protocol.PubKeyResponse{
		Type: "pubkeys",
		Keys: keys,
	}
	respData, _ := json.Marshal(resp)
	envelope := protocol.Message{
		Type: "pubkeys",
		Data: json.RawMessage(respData),
	}
	data, _ := json.Marshal(envelope)
	client.SendChan <- data

	return nil
}
